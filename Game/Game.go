package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	urlLib "net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	term "github.com/nsf/termbox-go"
)

type game struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Board    [3][3]int `json:"board"`
	Player   int       `json:"player"`
	Turn     int       `json:"turn"`
	Wins     [2]int    `json:"wins"`
	Players  []string  `json:"players"`
	Winner   int       `json:"winner"`
	WinTiles [3][2]int `json:"wintiles"`
}

type minimalGame struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Players []string `json:"players"`
}

var player int
var TicTacToe game

var width, height int

func main() {
	err := term.Init()
	if err != nil {
		panic(err)
	}

	defer clear()
	defer term.Close()
	width, height = term.Size()

	if width < 17 || height < 17 {
		fmt.Println("The terminal is too small! Please enlarge it to play.")
		width, height = waitForResize(17, 17)
	}

	go resizeListener()

	choice := menu([][2]string{{"1.", " Create"}, {"2.", " Join"}}, true)
	if choice == 0 {
		var keys string
		for {
			term.Clear(term.ColorWhite, term.ColorBlack)
			writeStr(fmt.Sprint("Name: ", string(keys)), (width/2)-len(fmt.Sprint("Name: ", string(keys))), (height / 2), term.ColorBlack, term.ColorWhite)
			key, char := keyListener()

			if key == 0 {
				keys = fmt.Sprint(keys, string(char))
			} else if key == term.KeySpace {
				keys = fmt.Sprint(keys, " ")
			} else if key == term.KeyBackspace && len(keys) > 0 {
				keys = popString(keys)
			} else if key == term.KeyEnter {
				break
			}
		}
		create(keys)
		boardSelect(boardChars(TicTacToe.Board), false)
		for TicTacToe.Winner == 0 {
			waitForUpdate(TicTacToe.Turn)
			row, column := boardSelect(boardChars(TicTacToe.Board), true)
			boardSelect(boardChars(TicTacToe.Board), false)
			update(row, column)
		}
	} else if choice == 1 {
		var gameStrings [][2]string
		games := all()
		for index, minimal := range games {
			gameStrings = append(gameStrings, [2]string{fmt.Sprintf("%v.", index), fmt.Sprintf(" %v          %v/2", minimal.Name, len(minimal.Players))})
		}
		index := menu(gameStrings, true)
		connect(games[index].ID)
	}
}

func resizeListener() {
	for {
		ev := term.PollEvent()
		if ev.Type == term.EventResize {
			term.Flush()
			width, height = term.Size()
		}
	}
}

func menu(choices [][2]string, newLine bool) int {
	term.Clear(term.ColorWhite, term.ColorBlack)
	width, height = term.Size()
	choice := 0

	var char int
	if !newLine {
		var totalLen int
		for _, str := range choices {
			totalLen += len(str[0])
			totalLen += len(str[1])
			totalLen += 3
		}
		char = 0 - (totalLen / 2)
	}

	for index, option := range choices {
		if newLine {
			char = 0 - (len(option[0]) + len(option[1]))
		}
		var fg, bg term.Attribute
		if choice == index {
			bg, fg = term.ColorWhite, term.ColorBlack
		} else {
			bg, fg = term.ColorBlack, term.ColorWhite
		}
		if newLine {
			writeStr(option[0], (width/2)+char, ((height/2)-(len(choices)-1)/2)+index, bg, fg)
			char += len(option[0])
			writeStr(option[1], (width/2)+char, ((height/2)-(len(choices)-1)/2)+index, term.ColorBlack, term.ColorWhite)
			char += len(option[1])
		} else {
			writeStr(option[0], (width/2)+char, height/2, bg, fg)
			char += len(option[0])
			writeStr(option[1], (width/2)+char, height/2, term.ColorBlack, term.ColorWhite)
			char += len(option[1])
			if index != len(choices)-1 {
				writeStr(" | ", (width/2)+char, height/2, term.ColorBlack, term.ColorWhite)
				char += 3
			}
		}
	}

	for {
		key := arrowListener()
		if key == 1 || key == 4 {
			if choice > 0 {
				choice = (choice - 1)
			} else {
				choice = len(choices) - 1
			}
		} else if key == 3 || key == 2 {
			if choice < len(choices)-1 {
				choice = (choice + 1)
			} else {
				choice = 0
			}
		} else if key == 0 {
			return choice
		}
		if choice < 0 {
			choice = -choice
		}
		if !newLine {
			var totalLen int
			for _, str := range choices {
				totalLen += len(str[0])
				totalLen += len(str[1])
				totalLen += 3
			}
			char = 0 - (totalLen / 2)
		}
		for index, option := range choices {
			if newLine {
				char = 0 - (len(option[0]) + len(option[1]))
			}
			var fg, bg term.Attribute
			if choice == index {
				bg, fg = term.ColorWhite, term.ColorBlack
			} else {
				bg, fg = term.ColorBlack, term.ColorWhite
			}
			if newLine {
				writeStr(option[0], (width/2)+char, ((height/2)-(len(choices)-1)/2)+index, bg, fg)
				char += len(option[0])
				writeStr(option[1], (width/2)+char, ((height/2)-(len(choices)-1)/2)+index, term.ColorBlack, term.ColorWhite)
				char += len(option[1])
			} else {
				writeStr(option[0], (width/2)+char, height/2, bg, fg)
				char += len(option[0])
				writeStr(option[1], (width/2)+char, height/2, term.ColorBlack, term.ColorWhite)
				char += len(option[1])
				if index != len(choices)-1 {
					writeStr(" | ", (width/2)+char, height/2, term.ColorBlack, term.ColorWhite)
					char += 3
				}
			}
		}
	}
}

func maxLen(list []string) int {
	max := 0
	for _, str := range list {
		if len(str) > max {
			max = len(str)
		}
	}
	return max
}

func gameFromPost(url string) game {
	var newGame game
	fakeJson, err := json.Marshal([]int{})
	if err != nil {
		panic(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(fakeJson))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&newGame)
	writeStr(fmt.Sprint(newGame), 0, 2, term.ColorBlack, term.ColorWhite)
	return newGame
}

func create(name string) {
	fmt.Println(fmt.Sprint("https://TicTacToe.jettbergthold.repl.co/create?name=", urlLib.QueryEscape(name)))
	TicTacToe = gameFromPost(fmt.Sprint("https://TicTacToe.jettbergthold.repl.co/create?name=", urlLib.QueryEscape(name)))
	player = 0
}

func connect(id int) {
	TicTacToe = gameFromPost(fmt.Sprint("https://TicTacToe.jettbergthold.repl.co/connect?gameid=", id))
	player = len(TicTacToe.Players) - 1
}

func update(row int, column int) {

}

func all() []minimalGame {
	var allGames []minimalGame
	resp, err := http.Get("https://TicTacToe.jettbergthold.repl.co/all")
	if err != nil {
		panic(err)
	}
	json.NewDecoder(resp.Body).Decode(&allGames)
	return allGames
}

func waitForUpdate(turn int) {
	for {
		var updated game
		resp, err := http.Get(fmt.Sprintf("https://TicTacToe.jettbergthold.repl.co/update?gameid=%v&uid=%v", TicTacToe.ID, TicTacToe.Players[player]))
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&updated)
		if updated.Turn > turn || updated.Player == player {
			TicTacToe = updated
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func boardChars(board [3][3]int) [3][3]string {
	var chars [3][3]string
	for row := 0; row < 3; row++ {
		for column := 0; column < 3; column++ {
			switch board[row][column] {
			case 0:
				chars[row][column] = "-"
			case 1:
				chars[row][column] = "X"
			case 2:
				chars[row][column] = "O"
			}
		}
	}
	return chars
}

func writeStr(str string, x int, y int, bg term.Attribute, fg term.Attribute) {
	lines := strings.Split(str, "\n")
	var runeLines [][]rune
	for _, line := range lines {
		runeLines = append(runeLines, []rune(line))
	}
	for lineIndex, line := range runeLines {
		for charIndex, char := range line {
			term.SetCell(x+charIndex, y+lineIndex, char, fg, bg)
		}
	}
	term.Flush()
}

func writeHighlight(str string, start [2]int, highlightPos [2][2]int, baseColor [2]term.Attribute, highColor [2]term.Attribute) {
	lines := strings.Split(str, "\n")
	var runeLines [][]rune
	for _, line := range lines {
		runeLines = append(runeLines, []rune(line))
	}
	for lineIndex, line := range runeLines {
		for charIndex, char := range line {
			if highlightPos[0][0] <= charIndex && charIndex <= highlightPos[1][0] && highlightPos[0][1] <= lineIndex && lineIndex <= highlightPos[1][1] {
				term.SetCell(start[0]+charIndex, start[1]+lineIndex, char, highColor[1], highColor[0])
			} else {
				term.SetCell(start[0]+charIndex, start[1]+lineIndex, char, baseColor[1], baseColor[0])
			}
		}
	}
	term.Flush()
}

func waitForResize(requiredWidth int, requiredHeight int) (int, int) {
	for {
		ev := term.PollEvent()
		if ev.Type == term.EventResize {
			if ev.Width >= requiredWidth && ev.Height >= requiredHeight {
				term.Flush()
				return term.Size()
			}
		}
	}
}

func arrowListener() int {
	for {
		ev := term.PollEvent()
		switch ev.Type {
		case term.EventKey:
			switch ev.Key {
			case term.KeyArrowUp:
				reset()
				return 1
			case term.KeyArrowDown:
				reset()
				return 3
			case term.KeyArrowLeft:
				reset()
				return 2
			case term.KeyArrowRight:
				reset()
				return 4
			case term.KeyEnter:
				return 0
			}
		case term.EventError:
			panic(ev.Err)
		}
	}
}

func keyListener() (term.Key, rune) {
	for {
		ev := term.PollEvent()
		if ev.Type == term.EventKey {
			return ev.Key, ev.Ch
		}
	}
}

func reset() {
	term.Sync()
}

func clear() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func popString(slice string) string {
	return slice[:len(slice)-1]
}

func boardSelect(chars [3][3]string, selecting bool) (int, int) {
	width, height = term.Size()
	selection := [2]int{0, 0}
	board := fmt.Sprintf(`     |     |     
  %v  |  %v  |  %v  
_____|_____|_____
     |     |     
  %v  |  %v  |  %v   
_____|_____|_____
     |     |     
  %v  |  %v  |  %v  
     |     |     `, chars[0][0], chars[0][1], chars[0][2], chars[1][0], chars[1][1], chars[1][2], chars[1][0], chars[1][1], chars[1][2])
	if selecting {
		writeHighlight(board, [2]int{(width / 2) - 8, (height / 2) - 8}, [2][2]int{{(selection[0] * 6) + 2, (selection[1] * 3) + 1}, {(selection[0] * 6) + 2, (selection[1] * 3) + 1}}, [2]term.Attribute{term.ColorBlack, term.ColorWhite}, [2]term.Attribute{term.ColorWhite, term.ColorBlack})
		for {
			key := arrowListener()
			if key == 1 && selection[1] > 0 {
				selection[1] -= 1
			} else if key == 3 && selection[1] < 2 {
				selection[1] += 1
			} else if key == 2 && selection[0] > 0 {
				selection[0] -= 1
			} else if key == 4 && selection[0] < 2 {
				selection[0] += 1
			} else if key == 0 {
				if chars[selection[0]][selection[1]] == "-" {
					return selection[0], selection[1]
				}
			}
			writeHighlight(board, [2]int{(width / 2) - 8, (height / 2) - 8}, [2][2]int{{(selection[0] * 6) + 2, (selection[1] * 3) + 1}, {(selection[0] * 6) + 2, (selection[1] * 3) + 1}}, [2]term.Attribute{term.ColorBlack, term.ColorWhite}, [2]term.Attribute{term.ColorWhite, term.ColorBlack})
		}
	} else {
		writeStr(board, (width/2)-8, (height/2)-8, term.ColorBlack, term.ColorWhite)
		return [2]int{-1, -1}
	}
}
