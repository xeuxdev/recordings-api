package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

type Album struct {
	ID     int64   `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float32 `json:"price"`
}

var db *sql.DB

func main() {

	cfg := mysql.Config{
		User:   os.Getenv("DBUSER"),
		Passwd: os.Getenv("DBPASS"),
		Net:    "tcp",
		Addr:   "127.0.0.1:3306",
		DBName: "recordings",
	}

	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())

	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()

	if pingErr != nil {
		log.Fatal(pingErr)
	}

	fmt.Println("Connected!")

	http.HandleFunc("/albums", addAlbumHandler)
	http.HandleFunc("/albums/artist", getAlbumsByArtistHandler)
	http.HandleFunc("/albums/get", getAlbumByIdHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func addAlbumHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}

	var album Album

	if err := json.NewDecoder(r.Body).Decode(&album); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := addAlbum(album)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	album.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(id)

}

func getAlbumsByArtistHandler(w http.ResponseWriter, r *http.Request) {

	var artist = r.URL.Query().Get("name")

	if artist == "" {
		http.Error(w, "Artist name missing", http.StatusBadRequest)
		return
	}

	album, err := getAlbumsByArtist(artist)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(album)

}

func getAlbumByIdHandler(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("albumId")
	albumId, err := strconv.ParseInt(id, 10, 64)

	if err != nil {
		http.Error(w, "Invalid album ID", http.StatusBadRequest)
		return
	}

	album, err := getAlbumById(albumId)

	if err != nil {
		if err.Error() == fmt.Sprintf("getAlbumById %d: no such album", id) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(album)

}

func getAlbumsByArtist(name string) ([]Album, error) {

	var albums []Album

	rows, err := db.Query("SELECT * FROM album WHERE artist = ?", name)
	if err != nil {
		return nil, fmt.Errorf("albums by artist %q: %v", name, err)
	}

	defer rows.Close()

	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, fmt.Errorf("albums by artist %q: %v", name, err)
		}
		albums = append(albums, alb)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("albums by artist %q: %v", name, err)
	}

	return albums, err
}

// albumByID queries for the album with the specified ID.

func getAlbumById(id int64) (Album, error) {
	var alb Album

	row := db.QueryRow("SELECT * FROM album WHERE id = ?", id)

	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("getAlbumById %d: no such album", id)
		}
		return alb, fmt.Errorf("getAlbumById %d: %v", id, err)
	}

	return alb, nil
}

// addAlbum adds the specified album to the database,
// returning the album ID of the new entry

func addAlbum(alb Album) (int64, error) {
	result, err := db.Exec("INSERT INTO album (title, artist, price) VALUES (?, ?, ?)", alb.Title, alb.Artist, alb.Price)

	if err != nil {
		return 0, fmt.Errorf("addAlbum: %v", err)
	}

	id, err := result.LastInsertId()

	if err != nil {
		return 0, fmt.Errorf("addAlbum: %v", err)
	}

	return id, nil
}
