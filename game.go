package main

import (
	"net/http"
	"html/template"
	"github.com/gorilla/websocket"
	"time"
	"math/rand"
	"log"
	"math"
	"sort"
	"github.com/ryanbradynd05/go-tmdb"
	"strconv"
)

const apiKey = "544bc5a0f64c7bdc16eedbf3a34e652c"				//api for the movies database

var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
}

var db *tmdb.TMDb

var Rooms map[string]*Room = make(map[string]*Room)				// global rooms
var broadcast = make(chan Message)           					// broadcast channel
var randomPeople []tmdb.PersonShort
//var randomMovies []tmdb.MovieShort

type TMDb struct {
	apiKey string
}

const MODE_SETUP = 1
const MODE_INGAME = 2
const MODE_BREAK = 3
const MODE_GAMEOVER = 4

type Room struct {
	ID        	string
	Players   	map[string]*Player
	Clients   	map[*websocket.Conn]bool
	Mode      	int
	HasAdmin  	bool
	MovieList   []*tmdb.MovieShort
	Questions 	[]*Question
	Ticker 		*time.Ticker
	Setup		*Setup
}

// Categories
const CATEGORY_POPULAR = 1
const CATEGORY_COMPANY = 2
const CATEGORY_GENERE = 3

type Setup struct {
	Category	int				`json:"category"`
	Options     []interface{}	`json:"options"`
}

// message types
const TYPE_UPDATE_ROOM = 1
const TYPE_START_GAME = 2
const TYPE_PLAYER_JOIN = 3
const TYPE_PLAYER_LEAVE = 4
const TYPE_PLAYER_ANSWER = 5
const TYPE_END_GAME = 6
const TYPE_CONTINUE_GAME = 7
const TYPE_BREAK_GAME = 8
const TYPE_GAME_MODE_CHANGE = 9
const TYPE_SETUP_CATEGORY = 10
const TYPE_READY = 11

type Update struct {
	Players		[]*Player 		`json:"players"`
	Current		*Player			`json:"current"`
	Mode 		int				`json:"mode"`
	Question	*Question		`json:"question"`
	Setup	 	*Setup			`json:"setup"`
	Type 		int 			`json:"type"`
	History		[]*Question		`json:"history"`
}

type Message struct {
	Room		*Room	`json:"-"`
	Type 		int		`json:"type"`
	Message  	string 	`json:"message"`
	Player		*Player	`json:"player"`
}

//player status:
const STATUS_PLAYING = 1
const STATUS_WATCHING = 2

type Player struct {
	ID 		string				`json:"id"`
	Name 	string				`json:"name"`
	Status  int	  				`json:"status"`
	Client  *websocket.Conn 	`json:"-"`
	Online	bool				`json:"online"`
	Admin	bool				`json:"admin"`
	Score	float32				`json:"score"`
	Guessed bool				`json:"guessed"`
}

type byScore []*Player
func (a byScore) Len() int           { return len(a) }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byScore) Less(i, j int) bool {
	return a[i].Score > a[j].Score
}

// question types
const TYPE_RATING = 1
const TYPE_CAST = 2
const TYPE_CHARACTER = 3
const TYPE_BUDJET = 4

type Question struct {
	ID     		int						`json:"id"`
	Movie		*tmdb.MovieShort		`json:"movie"`
	Style		int						`json:"style"`
	Expected	float32					`json:"expected"`
	Options		[]interface{}			`json:"options"`
	Answers		map[string]Answer		`json:"answers"`
	TimeLeft	int						`json:"timeleft"`
}

type Answer		struct {
	Player		*Player			`json:"player"`
	Value		float32			`json:"value"`
	Score		float32			`json:"score"`
}

type RatingQuestion struct{
	CorrectRating	float32
}

type PageVars struct {
	RoomID		string
	PlayerID	string
}

func generateRoomID() string{
	for {
		id := RandStringBytes(6)
		if _, found := Rooms[id] ; !found{
			return id
		}
	}
}
func generatePlayerID(room *Room) string{
	for {
		id := RandStringBytes(6)
		if _, found := room.Players[id] ; !found{
			return id
		}
	}
}

func RandStringBytes(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func loginNew(w http.ResponseWriter, newName string, room *Room) *Player{
	playerID := generatePlayerID(room);
	expiration := time.Now().Add(24 * time.Hour)
	cookie := http.Cookie{Name: "playerID", Value: playerID, Expires: expiration}
	http.SetCookie(w, &cookie)
	room.Players[playerID] = &Player{ID: playerID, Name: newName}
	return room.Players[playerID]
}

func logout(player *Player, room *Room){
	player.Client = nil
	room.Players[player.ID].Online = false
	if(player.Admin) {
		room.Players[player.ID].Admin = false
		room.HasAdmin = false;
	}
	var msg Message
	msg.Room = room
	msg.Player = player
	msg.Type = TYPE_PLAYER_LEAVE
	broadcast <- msg
}

func promoteAdmin(room *Room) *Player{
	for _,p := range room.Players{
		if p.Online{
			p.Admin = true
			room.HasAdmin = true;
			return p
		}
	}
	room.HasAdmin = false
	return nil
}

func connect(w http.ResponseWriter, r *http.Request){
	roomID := r.URL.Path[len("/connect/"):]
	// check room
	if _,found := Rooms[roomID]; !found {
		http.Redirect(w, r, "/", http.StatusFound) // no such room
	}
	t, _ := template.ParseFiles("static/connect.html")
	t.Execute(w, roomID)
}

func play(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Path[len("/play/"):]
	// check if room exist
	if _,found := Rooms[roomID]; !found {
		http.Redirect(w, r, "/", http.StatusFound) // no such room
	} else {
		var CurrentRoom *Room
		CurrentRoom = Rooms[roomID]
		// find player:
		var playerID string;
		if playerCookie, _ := r.Cookie("playerID") ; playerCookie == nil {
			if name := r.FormValue("inputName"); name != "" {
				// is connecting
				playerID = loginNew(w, name, CurrentRoom).ID
			} else {
				// need to connect first
				http.Redirect(w, r, "/connect/" + roomID, http.StatusFound)
			}
		} else {
			// if the player is not in the server
			if _,found := CurrentRoom.Players[playerCookie.Value]; !found{
				cookie := http.Cookie{Name: "playerID", Expires: time.Now()}
				http.SetCookie(w, &cookie)
				http.Redirect(w, r, "/connect/" + roomID, http.StatusFound)
			} else {
				playerID = playerCookie.Value
				// connect to room with cookie (do nothing)
			}
		}

		t, _ := template.ParseFiles("static/play.html")
		t.Execute(w, PageVars{RoomID: roomID, PlayerID: playerID})
	}
}

func create(w http.ResponseWriter, r *http.Request) {
	id := generateRoomID()
	Rooms[id] = &Room{ID: id, Players:  make(map[string]*Player),
		Mode: MODE_SETUP, HasAdmin: false, Setup: &Setup{Category: 1}}
	log.Println("Room was created - " + id)
	http.Redirect(w, r, "/play/" +  id, http.StatusFound)
}

func index(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("static/index.html")
	t.Execute(w, nil)
}

func checkAnswers(room *Room) bool{
	for _,p := range room.Players{
		if (p.Status == STATUS_PLAYING) && p.Online {
			if _,found := room.Questions[0].Answers[p.ID] ; !found{
				return false
			}
		}
	}
	return true
}
func fetchRandomPeople() []tmdb.PersonShort{
	// fetch random people
	randomPeople := []tmdb.PersonShort{}
	for i := 1; i < 6; i++{
		people, err := db.GetPersonPopular(map[string]string{"page": strconv.Itoa(i)})
		if err != nil{
			log.Println(err)
		}
		randomPeople = append(randomPeople, people.Results...)
	}
	return randomPeople
}
func generateQuestion(room *Room){
	// pop movie from the waiting movies
	style := rand.Intn(2) + 1
	movie := room.MovieList[0]
	room.Questions = append([]*Question{{ID: len(room.Questions), Movie: movie, Style: style,
		Answers: map[string]Answer{}, TimeLeft: 30}}, room.Questions...);
	switch style{
	case TYPE_RATING:
		room.Questions[0].Expected = movie.VoteAverage
	case TYPE_CAST:
		credits, err := db.GetMovieCredits(movie.ID, nil)
		if err != nil{
			log.Println("Cast Fetch: ", err)
		} else {
			room.Questions[0].Expected = float32(rand.Intn(4))
			// fetch random people
			var dontPeople map[int]bool = make(map[int]bool)
			for j := 0 ; j < len(credits.Cast) ; j++ {
				dontPeople[credits.Cast[j].ID] = true
			}
			for i := 0 ; i < len(credits.Cast) && i < 4; i++{
				if(i == int(room.Questions[0].Expected)){
					person := credits.Cast[rand.Intn(len(credits.Cast))]
					if person.ProfilePath != "" {
						room.Questions[0].Options = append(room.Questions[0].Options, person)
					} else {
						// fetch another
						i--
					}
				} else {
					person := randomPeople[rand.Intn(len(randomPeople))]
					if !dontPeople[person.ID] && person.ProfilePath != ""{
						room.Questions[0].Options = append(room.Questions[0].Options, randomPeople[rand.Intn(len(randomPeople))]);
						dontPeople[person.ID] = true
					} else {
						// fetch another
						i--
					}
				}
			}
		}
	}
	room.MovieList = room.MovieList[1:]
}
// this is a go routine that will prepare questions in the background for the game
func prepareQuestions(room *Room){
	for len(room.MovieList) > 0{
		generateQuestion(room)
		msg := Message{Room: room, Type: TYPE_READY}
		broadcast <- msg
	}
}

func resumeGame(room *Room){
	generateQuestion(room)
	for _, p := range room.Players {
		if (p.Online) {
			p.Status = STATUS_PLAYING
			p.Guessed = false
		}
	}
	room.Mode = MODE_INGAME
	// start timer!
	room.Ticker = time.NewTicker(1 * time.Second)
	go func() {
		for ; true; <-room.Ticker.C {
			room.Questions[0].TimeLeft--;
			if (room.Questions[0].TimeLeft < 0) {
				breakGame(room)
			}
		}
	}()
	upd := Update{Type: TYPE_GAME_MODE_CHANGE, Mode: room.Mode}
	if(len(room.Questions) > 0) {
		upd.Question = room.Questions[0]
	}
	updateRoom(room, &upd)
}
func fetchRandomMovies(n int, extraAttr map[string]string) []*tmdb.MovieShort{
	results := []tmdb.MovieShort{}
	for i := 1; i < 6; i++{
		attr := map[string]string{"sort_by": "popularity.asc",
			"vote_count.gte": "10","page": strconv.Itoa(i)}
		for k, v := range extraAttr {
			attr[k] = v
		}
		movies, err := db.DiscoverMovie(attr)
		if err != nil{
			log.Println(err)
		}
		results = append(results, movies.Results...)
	}
	movieList := []*tmdb.MovieShort{}
	for i := 0; i < n; i++ {
		movieList = append(movieList, &results[rand.Intn(len(results))])
	}
	return movieList
}
func setupGame(room *Room, n int, optionID int){
	var attr map[string]string = make(map[string]string)
	switch room.Setup.Category{
	case CATEGORY_POPULAR:
		attr["vote_count.gte"] = "10"
		room.MovieList = append(room.MovieList, fetchRandomMovies(n, attr)...)
	case CATEGORY_COMPANY:
		attr["with_companies"] = strconv.Itoa(optionID)
		room.MovieList = append(room.MovieList, fetchRandomMovies(n, attr)...)
	case CATEGORY_GENERE:
		attr["with_genres"] = strconv.Itoa(optionID)
	}
	room.MovieList = append(room.MovieList, fetchRandomMovies(n, attr)...)
	// prepare questions in the background
	//go prepareQuestions(room)
}
func breakGame(room *Room){
	room.Ticker.Stop()
	for _, answer := range room.Questions[0].Answers {
		answer.Player.Score += answer.Score
	}
	if len(room.MovieList) > 0 {
		room.Mode = MODE_BREAK
		upd := Update{Type: TYPE_BREAK_GAME, Mode: MODE_BREAK, Question: room.Questions[0], Players: listPlayers(room)}
		updateRoom(room, &upd)
	} else {
		room.Mode = MODE_GAMEOVER
		// add aditional mode for finished games
		upd := Update{Type:TYPE_END_GAME, Mode: MODE_GAMEOVER, History: room.Questions[:10], Players: listPlayers(room)}
		updateRoom(room, &upd)
	}
}

func calculateScore(question *Question, answer float32) float32{
	var score float32 = 0
	switch question.Style {
	case TYPE_RATING:
		score = float32(10 - (math.Abs(float64(question.Expected) - float64(answer))) * 1.5)
		if score == 10 {
			score = 20
		} else if score < 6{
			score = 0
		}
	case TYPE_CAST:
		if question.Expected == answer{
			score = 10
		}
	}
	return float32(score)
}

func listPlayers(room *Room) []*Player{
	// player list sorted by score
	players := []*Player{}
	for _, player := range room.Players {
		players = append(players, player)
	}
	sort.Sort(byScore(players))
	return players
}

func updatePlayer(room *Room, player *Player, upd *Update){
	err := player.Client.WriteJSON(upd)
	if err != nil {
		log.Printf("error: %v", err)
		logout(player, room)
	}
}

func updateRoom(room *Room, upd *Update){
	for _, p := range room.Players {
		//if player is connected
		upd.Current = p
		if p.Online {
			err := p.Client.WriteJSON(upd)
			if err != nil {
				log.Printf("error: %v", err)
				logout(p, room)
			}
		}
	}
}

func handleUpdates(){
	for {
		// Grab the next message from the broadcast channel
		msg := <- broadcast
		room := msg.Room
		player := msg.Player
		log.Println(msg.Type)
		switch msg.Type {
		case TYPE_PLAYER_JOIN:
			players := listPlayers(room)
			// send other players an update to inform player joined
			updOthers := Update{Type: TYPE_PLAYER_JOIN}
			updOthers.Players = players
			updateRoom(room, &updOthers)

			// send player all initial info
			updCurrent := Update{Type: TYPE_UPDATE_ROOM}
			updCurrent.Mode = room.Mode
			if (room.Mode == MODE_INGAME || room.Mode == MODE_BREAK) {
				updCurrent.Question = room.Questions[0]
			}
			if (room.Mode == MODE_GAMEOVER){
				updCurrent.History = room.Questions[:10]
			}
			updCurrent.Setup = room.Setup
			// update current player
			updatePlayer(room, player, &updCurrent)

		case TYPE_PLAYER_LEAVE:
			upd := Update{Type: TYPE_PLAYER_LEAVE}
			if (!room.HasAdmin) {
				admin := promoteAdmin(room)
				if (admin != nil) {
					updatePlayer(room, admin, &Update{Current: admin})
				}
			}
			players := listPlayers(room)
			upd.Players = players
			updateRoom(room, &upd)

		case TYPE_SETUP_CATEGORY:
			cat, err := strconv.Atoi(msg.Message)
			if err != nil {
				log.Println(err)
			} else {
				room.Setup.Category = cat
				room.Setup.Options = room.Setup.Options[:0]
				switch(room.Setup.Category) {
				case CATEGORY_COMPANY:
					for i := 1; i < 13; i++ {
						company, err := db.GetCompanyInfo(i, nil)
						if err != nil {
							log.Println(err)
						} else {
							room.Setup.Options = append(room.Setup.Options, company)
						}
					}
				}
				updateRoom(room, &Update{Setup: room.Setup})
			}

		case TYPE_PLAYER_ANSWER:
			//check if player didn't answer yet
			if len(room.Questions) > 0 {
				if _, found := room.Questions[0].Answers[player.ID]; !found {
					value, err := strconv.ParseFloat(msg.Message, 32)
					if (err != nil) {
						log.Println(err)
					} else {
						room.Questions[0].Answers[player.ID] = Answer{Player: player,
							Value: float32(value), Score: calculateScore(room.Questions[0], float32(value))}
						player.Guessed = true
						upd := Update{Type: TYPE_PLAYER_ANSWER, Question: room.Questions[0], Current: player}
						updatePlayer(room, player, &upd)
					}
				}
				// if everyone answered
				if checkAnswers(room) {
					breakGame(room)
				}
			}

		case TYPE_START_GAME:
			if (player.Admin) {
				if optionID, err := strconv.Atoi(msg.Message); err == nil {
					setupGame(room, 10, optionID)
				} else {
					setupGame(room, 10, 0)
				}
				resumeGame(room)
			}

		case TYPE_CONTINUE_GAME:
			if (player.Admin) {
				resumeGame(room)
			}
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Path[4:10]
	playerID := r.URL.Path[10:16]
	// upgrade request to a web socket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	//close connection when function returns
	defer ws.Close();
	room := Rooms[roomID]
	player := room.Players[playerID]
	player.Client = ws
	player.Online = true
	if room.Mode == MODE_INGAME{
		player.Status = STATUS_WATCHING
	} else {
		player.Status = STATUS_PLAYING
	}
	if !room.HasAdmin{
		player.Admin = true
		room.HasAdmin = true
	}
	joinUPD := Message{Type: TYPE_PLAYER_JOIN, Room: room, Player: player}
	broadcast <- joinUPD
	// listen for messages from the current player
	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			logout(player, room)
			break
		}
		// apply the current room to the message
		msg.Room = room
		msg.Player = player
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

func main() {
	db = tmdb.Init(apiKey)
	randomPeople = fetchRandomPeople()

	http.HandleFunc("/ws/", handleConnections)
	http.HandleFunc("/play/", play)
	http.HandleFunc("/connect/", connect)
	http.HandleFunc("/create", create)
	http.HandleFunc("/", index)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Println("http server started on :8080")
	go handleUpdates()
	if err := http.ListenAndServe(":8080", nil) ; err != nil{
		log.Fatal("ListenAndServe: ", err)
	}

}