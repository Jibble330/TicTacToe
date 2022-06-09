package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type game struct {
	ID      int       `json:"id"`
	Name    string    `json:"name"`
	Board   [3][3]int `json:"board"`
	Player  int       `json:"player"`
	Turn    int       `json:"turn"`
	Wins    [2]int    `json:"wins"`
	Players int       `json:"players"`
	Winner  int       `json:"winner"`
}

var games []game

//Server functions
func sendGame(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if catch(err) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	gameIndex, err := findGame(id)
	if catch(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.IndentedJSON(http.StatusOK, games[gameIndex])
}

func update(c *gin.Context) {
	//Parse arguments from url
	id, idErr := strconv.Atoi(c.Query("id"))
	player, playerErr := strconv.Atoi(c.Query("player"))
	row, rowErr := strconv.Atoi(c.Query("row"))
	column, columnErr := strconv.Atoi(c.Query("column"))
	//Check for errors
	if catch(idErr) || catch(playerErr) || catch(rowErr) || catch(columnErr) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	c.Status(http.StatusOK)
	gameIndex, err := findGame(id)
	if catch(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
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
		Players: 1,
		Winner:  0,
	}
	games = append(games, newGame)
	c.IndentedJSON(http.StatusCreated, newGame)
	fmt.Println(games)
}

func close(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if catch(err) {
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}
	index, err := findGame(id)
	if catch(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	games = remove(games, index)
}

func connect(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if catch(err) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	gameIndex, err := findGame(id)
	if catch(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if games[gameIndex].Players >= 2 {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	games[gameIndex].Players = 2
	//Need random string for id
	c.Writer.WriteString(fmt.Sprintf("{\"id\": %v}", random))
}

func reset(c *gin.Context) {

}

func all(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, games)
}

//Game functions
func win(board [3][3]int) int {
	//Check rows
	for i := 0; i < 3; i++ {
		if sum(board[i]) == 3 && !contains(2, board[i]) {
			return 1
		} else if sum(board[i]) == 6 {
			return 2
		}
	}
	//Check columns
	for i := 0; i < 3; i++ {
		column := [3]int{board[0][i], board[1][i], board[2][i]}
		if sum(column) == 3 && !contains(2, column) {
			return 1
		} else if sum(column) == 6 {
			return 2
		}
	}
	//Check for diagonals
	if board[0][0] == 1 && board[1][1] == 1 && board[2][2] == 1 {
		return 1
	} else if board[0][0] == 2 && board[1][1] == 2 && board[2][2] == 2 {
		return 2
	} else if board[2][0] == 1 && board[1][1] == 1 && board[0][2] == 1 {
		return 1
	} else if board[2][0] == 2 && board[1][1] == 2 && board[0][2] == 2 {
		return 2
	}
	return 0
}

func gameUpdate(query game) game {
	query.Turn++
	query.Player = 1 - query.Player
	won := win(query.Board)
	query.Winner = won
	if won != 0 {
		query.Wins[won-1]++
	}
	return query
}

func findGame(id int) (int, error) {
	for i := 0; i < len(games); i++ {
		if games[i].ID == id {
			return i, nil
		}
	}
	return -1, errors.New("Game with specified ID not found")
}

//utils
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

func remove(slice []game, index int) []game {
	return append(slice[:index], slice[index+1:]...)
}

func main() {
	router := gin.Default()
	router.GET("/update", sendGame)
	router.GET("/all", all)
	router.POST("/update", update)
	router.POST("/reset", reset)
	router.POST("/create", create)
	router.POST("/close")
	router.POST("/connect", connect)
	router.Run("0.0.0.0:8080")
}
