package main

import (
	bitcaskgo "bitcask-go"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

var db *bitcaskgo.DB

func init() {
	var err error
	options := bitcaskgo.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-http")
	options.DirPath = dir
	db, err = bitcaskgo.Open(options)
	if err != nil {
		return
	}
}
func handlePut(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var data map[string]string
	if err := json.NewDecoder(request.Body).Decode(&data); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	for k, v := range data {
		if err := db.Put([]byte(k), []byte(v)); err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			log.Printf("failed to put key:%s,value:%s", k, v)
			return
		}
	}
}
func handleGet(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := request.URL.Query().Get("key")
	value, err := db.Get([]byte(key))
	if err != nil && err != bitcaskgo.ErrKeyIsUnfound {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		log.Printf("failed to get key:%s from db", key)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(string(value))
}
func handleDelete(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodDelete {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := request.URL.Query().Get("key")
	err := db.Delete([]byte(key))
	if err != nil && err != bitcaskgo.ErrKeyIsUnfound {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		log.Printf("failed to delete key:%s from db", key)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode("OK")
}
func handleListKeys(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys := db.ListKeys()
	writer.Header().Set("Content-Type", "application/json")
	var result []string
	for _, k := range keys {
		result = append(result, string(k))
	}
	_ = json.NewEncoder(writer).Encode(result)
}

func handleStat(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stat := db.Stat()
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(stat)
}

func main() {
	//注册处理方法
	http.HandleFunc("/bitcaskgo/put", handlePut)
	http.HandleFunc("/bitcaskgo/get", handleGet)
	http.HandleFunc("/bitcaskgo/delete", handleDelete)
	http.HandleFunc("/bitcaskgo/listKeys", handleListKeys)
	http.HandleFunc("/bitcaskgo/stat", handleStat)
	_ = http.ListenAndServe("localhost:8080", nil)
}
