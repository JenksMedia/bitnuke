package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/unixvoid/glogger"
	"golang.org/x/crypto/sha3"
	"gopkg.in/redis.v3"
)

func handlerdynamic(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	vars := mux.Vars(r)
	fdata := vars["fdata"]

	// hash the token that is passed
	hash := sha3.Sum512([]byte(fdata))
	hashstr := fmt.Sprintf("%x", hash)

	// get the client's ip
	ip := strings.Split(r.RemoteAddr, ":")[0]

	// set client as localhost if it comes from localhost
	if ip == "[" {
		ip = "localhost"
	}

	// pull the client's real header if proxied. (if X-Forwarded-For is set)
	realIp := r.Header.Get("X-Forwarded-For")
	if realIp != "" {
		ip = realIp
	}

	// try and pull the data from redis
	val, err := redisClient.Get(hashstr).Result()
	filename, err := redisClient.Get(fmt.Sprintf("fname:%s", hashstr)).Result()
	if err != nil {
		// handle the error if the token does not exist
		glogger.Debug.Printf("data does not exist %s :: from: %s\n", fdata, ip)
		fmt.Fprintf(w, "token not found")
	} else {
		// serve up the content to the client
		glogger.Debug.Printf("Responsing to %s :: from: %s\n", fdata, ip)

		decodeVal, _ := base64.StdEncoding.DecodeString(val)

		file, _ := os.Create("tmpfile")
		io.WriteString(file, string(decodeVal))
		file.Close()

		// dont add the filename header to links
		if filename != "bitnuke:link" {
			finalFname := fmt.Sprintf("INLINE; filename=%s", filename)
			w.Header().Set("Content-Disposition", finalFname)
		}
		http.ServeFile(w, r, "tmpfile")
		os.Remove("tmpfile")
	}
}
