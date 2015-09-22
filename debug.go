package main

import "os"

var DEBUG = os.Getenv("DEBUG") != ""
