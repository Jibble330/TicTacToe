package main

import (
    "errors"
    "fmt"
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/rs/xid"
)

type game struct {
    ID      int        `json:"id"`
    Name    string     `json:"name"`
    Board   [3][3]int  `json:"board"`
    Player  int        `json:"player"`
    Turn    int        `json:"turn"`
    Wins    [2]int     `json:"wins"`
    Players []string   `json:"players"`
    Connected []bool
    Winner  int        `json:"winner"`
    WinTiles [3][2]int `json:"wintiles"`
}

var games []game

//Server functions
func sendGame(c *gin.Context) {
    //Add uid checker to this
    gameId, gameErr := strconv.Atoi(c.Query("gameid"))
    uid := c.Query("uid")
    if catch(gameErr) || uid == ""{
        c.Status(http.StatusBadRequest)
        return
    }
    gameIndex, err := findGame(gameId)
    if catch(err) {
        c.Status(http.StatusNotFound)
        return
    }
    if getPlayer(games[gameIndex], uid) == 0 {
        c.Status(http.StatusForbidden)
        return
    }
    c.IndentedJSON(http.StatusOK, games[gameIndex])
}

func update(c *gin.Context) {
    //Parse arguments from url
    gameId, gameErr := strconv.Atoi(c.Query("gameid"))
    uid := c.Query("uid")
    row, rowErr := strconv.Atoi(c.Query("row"))
    column, columnErr := strconv.Atoi(c.Query("column"))
    
    if catch(gameErr) || catch(rowErr) || catch(columnErr) {
        c.Status(http.StatusBadRequest)
        return
    }

    c.Status(http.StatusOK)
    gameIndex, err := findGame(gameId)
    if catch(err) {
        c.Status(http.StatusNotFound)
        return
    }
    
    player := getPlayer(games[gameIndex], uid)
    games[gameIndex].Board[row][column] = player
    games[gameIndex] = gameUpdate(games[gameIndex])
    c.IndentedJSON(http.StatusOK, games[gameIndex])
}

func create(c *gin.Context) {
    name := c.Query("name")
    fmt.Println(name)
    var id int
    fmt.Println(len(games))
    if len(games) == 0 {
        id = 0
    } else {
        id = (games[len(games)-1].ID + 1)
    }
    fmt.Println(id)
    newGame := game{
        ID:      id,
        Name:    name,
        Board:   [3][3]int{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}},
        Player:  0,
        Turn:    1,
        Wins:    [2]int{0, 0},
        Players: []string{randId()},
        Connected: []bool{true},
        Winner:  0,
    }
    games = append(games, newGame)
    c.IndentedJSON(http.StatusCreated, newGame)
    fmt.Println(games)
}

func destroy(c *gin.Context) {
    gameId, err := strconv.Atoi(c.Query("gameid"))
    if catch(err) {
        c.Status(http.StatusBadGateway)
        return
    }
    index, err := findGame(gameId)
    if catch(err) {
        c.Status(http.StatusNotFound)
        return
    }
    games = removeGame(games, index)
}

func connection(c *gin.Context) {
    gameId, err := strconv.Atoi(c.Query("gameid"))
    if catch(err) {
        c.Status(http.StatusBadRequest)
        return
    }

    gameIndex, err := findGame(gameId)
    if catch(err) {
        c.Status(http.StatusNotFound)
        return
    }

    uid := c.Query("uid")
    player := getPlayer(games[gameIndex], uid)
    if uid == "" || player == 0 {
        c.Status(http.StatusBadRequest)
        return
    }

    games[gameIndex].Connected[player-1] = true
}

func connect(c *gin.Context) {
    gameId, err := strconv.Atoi(c.Query("gameid"))
    if catch(err) {
        c.Status(http.StatusBadRequest)
        return
    }
    gameIndex, err := findGame(gameId)
    if catch(err) {
        c.Status(http.StatusNotFound)
        return
    }
    if len(games[gameIndex].Players) >= 2 {
        c.Status(http.StatusForbidden)
        return
    }
    
    uid := randId() //Generate random id to identify player 2
    games[gameIndex].Turn += 1
    games[gameIndex].Players = append(games[gameIndex].Players, uid)
    games[gameIndex].Connected = append(games[gameIndex].Connected, true)
    c.IndentedJSON(http.StatusOK, games[gameIndex])
}

func reset(c *gin.Context) {
    gameId, err := strconv.Atoi(c.Query("gameid"))
    if catch(err) {
        c.Status(http.StatusBadRequest)
        return
    }
    uid := c.Query("uid")
    if uid == "" {
        c.Status(http.StatusBadRequest)
        return
    }
    gameIndex, err := findGame(gameId)
    if catch(err) {
        c.Status(http.StatusNotFound)
        return
    }
    games[gameIndex].Board = [3][3]int{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}}
    games[gameIndex].Turn = 0
    games[gameIndex].Winner = 0
    games[gameIndex].Player = 0
    c.IndentedJSON(http.StatusOK, games[gameIndex])
}

func disconnect(c *gin.Context) {
    gameId, err := strconv.Atoi(c.Query("gameid"))
    if catch(err) {
        c.Status(http.StatusBadRequest)
        return
    }

    gameIndex, err := findGame(gameId)
    if catch(err) {
        c.Status(http.StatusNotFound)
        return
    }

    uid := c.Query("uid")
    player := getPlayer(games[gameIndex], uid)
    if uid == "" || player == 0 {
        c.Status(http.StatusBadRequest)
        return
    }

    disconnectPlayer(gameIndex, player)
    c.Status(http.StatusOK)
}

func all(c *gin.Context) {
    c.IndentedJSON(http.StatusOK, games)
}

//Game functions
func win(board [3][3]int) (int, [3][2]int) {
    //Check rows
    for i := 0; i < 3; i++ {
        if sum(board[i]) == 3 && !contains(2, board[i]) {
            return 1, [3][2]int{{i, 0}, {i, 1}, {i, 2}}
        } else if sum(board[i]) == 6 {
            return 2, [3][2]int{{i, 0}, {i, 1}, {i, 2}}
        }
    }
    //Check columns
    for i := 0; i < 3; i++ {
        column := [3]int{board[0][i], board[1][i], board[2][i]}
        if sum(column) == 3 && !contains(2, column) {
            return 1, [3][2]int{{0, i}, {1, i}, {2, i}}
        } else if sum(column) == 6 {
            return 2, [3][2]int{{0, i}, {1, i}, {2, i}}
        }
    }
    //Check for diagonals
    if board[0][0] == 1 && board[1][1] == 1 && board[2][2] == 1 {
        return 1, [3][2]int{{0, 0}, {1, 1}, {2, 2}}
    } else if board[0][0] == 2 && board[1][1] == 2 && board[2][2] == 2 {
        return 2, [3][2]int{{0, 0}, {1, 1}, {2, 2}}
    } else if board[2][0] == 1 && board[1][1] == 1 && board[0][2] == 1 {
        return 1, [3][2]int{{2, 0}, {1, 1}, {0, 2}}
    } else if board[2][0] == 2 && board[1][1] == 2 && board[0][2] == 2 {
        return 2, [3][2]int{{2, 0}, {1, 1}, {0, 2}}
    }
    return 0, [3][2]int{{-1, -1}, {-1, -1}, {-1, -1}}
}

func gameUpdate(query game) game {
    query.Turn++
    query.Player = 1 - query.Player
    won, winTiles := win(query.Board)
    if won != 0 {
        query.Winner = won
        query.Wins[won-1]++
        query.WinTiles = winTiles
    }
    return query
}

func findGame(id int) (int, error) {
    for i := 0; i < len(games); i++ {
        if games[i].ID == id {
            return i, nil
        }
    }
    return -1, errors.New("game with specified id not found")
}

//Utility functions
func sum(arr [3]int) (summed int) {
    for i := 0; i < len(arr); i++ {
        summed += arr[i]
    }
    return summed
}

func contains(query int, arr [3]int) bool {
    for _, elem := range arr {
        if elem == query {
            return true
        }
    }
    return false
}

func catch(err error) bool {
    return err != nil
}

func removeGame(slice []game, index int) []game {
    return append(slice[:index], slice[index+1:]...)
}

func randId() string {
    uid := xid.New().String()
    duplicate := false
    for _, unique := range games {
        if getPlayer(unique, uid) != 0 {
            duplicate = true
        }
    }
    for duplicate {
        uid := xid.New().String()
        for _, unique := range games {
            if getPlayer(unique, uid) != 0 {
                duplicate = true
            }
        }
    }
    return uid
}

func getPlayer(unique game, uid string) int {
    for player, id := range unique.Players {
        if uid == id {
            return player+1
        }
    }
    return 0
}

func disconnectPlayer(gameIndex int, player int) {
    fmt.Println("Player disconnected!")
    if len(games[gameIndex].Players) < 2 {
        games = removeGame(games, gameIndex)
    } else {        
        resetGame := game{
            ID:      games[gameIndex].ID,
            Name:    games[gameIndex].Name,
            Board:   [3][3]int{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}},
            Player:  0,
            Turn:    1,
            Wins:    [2]int{0, 0},
            Players: []string{games[gameIndex].Players[0-(player-1)]},
            Connected: []bool{games[gameIndex].Connected[0-(player-1)]},
            Winner:  0,
        }
        games[gameIndex] = resetGame
    }
}

func checkConn() {
    for {
        for gameIndex, Game := range games {
            for playerIndex, connected := range Game.Connected {
                if !connected {
                    disconnectPlayer(gameIndex, playerIndex)
                } else {
                    games[gameIndex].Connected[playerIndex] = false
                }
            }
            time.Sleep(5*time.Second)
        }
    }
}

func main() {
    gin.SetMode(gin.DebugMode)
    router := gin.Default()
    router.GET("/all", all)
    router.GET("/update", sendGame)
    router.POST("/update", update)
    router.POST("/reset", reset)
    router.POST("/create", create)
    router.POST("/destroy", destroy)
    router.POST("/connect", connect)
    router.POST("/connection", connection)
    router.POST("/disconnect", disconnect)

    go checkConn()

    router.Run("0.0.0.0:8080")
}
