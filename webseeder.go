package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/go-multierror"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("webseeder.cfg")    // name of config file (without extension)
	viper.SetConfigType("properties")       // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("$HOME/.webseeder") // call multiple times to add many search paths
	viper.AddConfigPath(".")                // optionally look for config in the working directory
	err := viper.ReadInConfig()             // Find and read the config file
	if err != nil {                         // Handle errors reading the config file
		log.Fatalf("Fatal error config file: %s \n", err)
	}

	if err := validateConfig(); err != nil {
		log.Fatalf("Fatal configuration error: %s \n", multierror.Flatten(err))
	}
}

var (
	mainPort string
	useTLS   = false
	tlsPort  string
	tlsCert  string
	tlsKey   string

	htAuthUser string
	htAuthPass string

	staticFilePath   string
	torrentsFilePath string
)

func validateConfig() error {
	var result *multierror.Error
	mainPort = viper.GetString("port")
	if len(mainPort > 0) {
		portint, err := strconv.Atoi(mainPort)
		if err != nil {
			repErr := multierror.Prefix(err, "Port not valid number")
			result = multierror.Append(result, repErr)
		}
		if portint <= 0 || portint > 65535 {
			result = multierror.Append(result, errors.New("Port outside valid range"))
		}
	}
	tlsPort = viper.GetString("tlsport")
	if len(tlsPort) > 0 {
		useTLS = true
		tlsportint, err := strconv.Atoi(tlsPort)
		if err != nil {
			repErr := multierror.Prefix(err, "TLS Port not valid number")
			result = multierror.Append(result, repErr)
		}
		if tlsportint <= 0 || tlsportint > 65535 {
			result = multierror.Append(result, errors.New("TLS Port outside valid range"))
		}

		tlsCert = viper.GetString("tlscert")
		_, err = os.Stat(tlsCert)
		if os.IsNotExist(err) {
			result = multierror.Append(result, errors.New("TLS Cert file not found"))
		}

		tlsKey = viper.GetString("tlskey")
		_, err = os.Stat(tlsKey)
		if os.IsNotExist(err) {
			result = multierror.Append(result, errors.New("TLS Key file not found"))
		}
	}

	if len(viper.GetString("username")) < 1 {
		log.Println("Warning: No username set.")
	} else {
		htAuthUser = viper.GetString("username")
	}
	if len(viper.GetString("password")) < 1 {
		log.Println("Warning: No password set.")
	} else {
		htAuthPass = viper.GetString("password")
	}

	staticFilePath = viper.GetString("filepath")
	if len(staticFilePath) < 1 {
		result = multierror.Append(result, errors.New("FilePath not set"))
	}
	fi, err := os.Stat(staticFilePath)
	if os.IsNotExist(err) {
		result = multierror.Append(result, errors.New("FilePath doesn't exist"))
	} else if fi.Mode(); !fi.IsDir() {
		result = multierror.Append(result, errors.New("FilePath is not a valid directory"))
	}

	torrentsFilePath = viper.GetString("torrentsfilepath")
	if len(torrentsFilePath) < 1 {
		log.Println("Warning: No torrent cache path set.")
	} else {
		fi, err := os.Stat(torrentsFilePath)
		if os.IsNotExist(err) {
			result = multierror.Append(result, errors.New("Torrents path doesn't exist"))
		} else if fi.Mode(); !fi.IsDir() {
			result = multierror.Append(result, errors.New("Torrents path is not a valid directory"))
		}
	}

	return result.ErrorOrNil()
}

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hello)
	gl := e.Group("/list")
	gf := e.Group("/files")
	if len(htAuthUser) > 0 && len(htAuthPass) > 0 {
		gf.Use(middleware.BasicAuth(authMidHandler))
		gl.Use(middleware.BasicAuth(authMidHandler))
	}
	gf.Static("/", staticFilePath)

	if len(torrentsFilePath) > 0 {
		log.Println("using listing")
		gl.GET("", listHandler)
		gl.GET("/", listHandler)
	}

	// Start server
	if useTLS {
		go func() {
			e.Logger.Fatal(e.StartTLS(":"+tlsPort, tlsCert, tlsKey))
		}()
	}
	e.Logger.Fatal(e.Start(":" + mainPort))
}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "github.com/zvodd/webseeder")
}

// list
func listHandler(c echo.Context) error {

	var files []string
	err := filepath.Walk(torrentsFilePath, func(path string, info os.FileInfo, err error) error {
		p, _ := filepath.Rel(torrentsFilePath, path)
		if p != "." {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		c.String(http.StatusFailedDependency, "nope")
	}
	return c.JSON(http.StatusAccepted, files)

}

func authMidHandler(username, password string, c echo.Context) (bool, error) {
	if username == htAuthUser && password == htAuthPass {
		return true, nil
	}
	return false, nil
}
