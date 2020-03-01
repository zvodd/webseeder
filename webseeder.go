package main

import (
	"errors"
	"fmt"
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

func init(){
	viper.SetConfigName("webseeder.cfg") // name of config file (without extension)
	viper.SetConfigType("properties") // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("$HOME/.webseeder")  // call multiple times to add many search paths
	viper.AddConfigPath(".")               // optionally look for config in the working directory
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	
	if err := verifyConfig(); err != nil {
		panic(fmt.Errorf("Fatal configuration error: %s \n", multierror.Flatten(err)))
	}
}

func verifyConfig() error{
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
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		result = multierror.Append(result, errors.New("FilePath doesn't exist"))
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
  return c.String(http.StatusOK, "Hello, World!")
}

func filesHandler(c echo.Context) error {
	fileName := c.Param("fileName")
	fileLocation := filepath.Join(viper.GetString("filepath"), fileName)
	return c.File(fileLocation)
	// return c.String(http.StatusOK, "/users/:"+id)
}


func authMidHandler(username, password string, c echo.Context) (bool, error) {
	if username == viper.GetString("username") && password == viper.GetString("password") {
		return true, nil
	}
	return false, nil
}