package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/hashicorp/go-multierror"

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
)

func init(){
	viper.SetConfigName("webseeder.cfg") // name of config file (without extension)
	viper.SetConfigType("properties") // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("$HOME/.webseeder")  // call multiple times to add many search paths
	viper.AddConfigPath(".")               // optionally look for config in the working directory
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil { // Handle errors reading the config file
		log.Fatalf("Fatal error config file: %s \n", err)
	}
	
	if err := validateConfig(); err != nil {
		log.Fatalf("Fatal configuration error: %s \n", multierror.Flatten(err))
	}
}

func validateConfig() error{
	var result *multierror.Error
	port := viper.GetString("port")
	portint, err := strconv.Atoi(port);
	if err != nil {
		repErr := multierror.Prefix(err, "Port not valid number") 
		result = multierror.Append(result, repErr)
	}
	if portint <= 0 || portint > 65535 {
		result = multierror.Append(result, errors.New("Port outside valid range"))
	}

	if len(viper.GetString("username")) <1{
		log.Println("Warning: No username set.")
	}
	if len(viper.GetString("password")) <1{
		log.Println("Warning: No password set.")
	}
	
	filePath := viper.GetString("filepath")
	if len(filePath) < 1 {
		result = multierror.Append(result, errors.New("FilePath not set"))
	}
	fi, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		result = multierror.Append(result, errors.New("FilePath doesn't exist"))
	}else if fi.Mode(); !fi.IsDir(){
		result = multierror.Append(result, errors.New("FilePath is not a valid directory"))
	}


	rtcp := viper.GetString("rtorrent_cache_path")
	if len(rtcp) < 1{
		log.Println("Warning: No torrent cache path set.")
	}else{
		fi, err := os.Stat(rtcp)
		if os.IsNotExist(err) {
			result = multierror.Append(result, errors.New("rtorrent_cache_path doesn't exist"))
		}else if fi.Mode(); !fi.IsDir(){
			result = multierror.Append(result, errors.New("rtorrent_cache_path is not a valid directory"))
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
  g := e.Group("/files")
  if (len(viper.GetString("username")) > 0 && len(viper.GetString("password")) > 0){
	  g.Use(middleware.BasicAuth(authMidHandler))
  }
  g.Static("/", viper.GetString("filepath"))

  // Start server
  e.Logger.Fatal(e.Start(":"+viper.GetString("port")))
}

// Handler
func hello(c echo.Context) error {
  return c.String(http.StatusOK, "github.com/zvodd/webseeder")
}


// list
func listHandler(c echo.Context) error {
	// rtcp := viper.GetString("rtorrent_cache_path")
	
	return nil
}

func authMidHandler(username, password string, c echo.Context) (bool, error) {
	if username == viper.GetString("username") && password == viper.GetString("password") {
		return true, nil
	}
	return false, nil
}