package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"encoding/json"
)

type jsonObj struct {
	Object ObjectType
}

type ObjectType struct {
	api
}

func main(configPath string) {
	fmt.Println("Setting up the test")
	file, e := ioutil.ReadFile(configPath)

	if e != nil {
		fmt.Printf("File read error: %v\n", e)
		os.Exit(1)
	}

	var configVars jsonObj
	json.Unmarshal(file, &configVars)
	fmt.Printf("JSON: %v\n", configVars)
}
