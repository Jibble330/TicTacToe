package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    urlLib "net/url"
    "os/exec"
    "strings"
    "time"
    "os"
    "runtime"

    keyboard "github.com/eiannone/keyboard"
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

type Cell struct {
    Ch rune
    Fg term.Attribute
    Bg term.Attribute
    X, Y int
}

//Globals
var player int
var TicTacToe game
var width, height int
var connected bool
var listening bool

func main() {
    connected = false
    listening = false
    err := term.Init()
    if err != nil {
        panic(err)
    }
    defer clear()
    defer term.Close()

    defer Exit()

    width, height = term.Size()
    if width < 17 || height < 17 {
        writeStr("The terminal is too small! Please enlarge it to play.", (width/2)-26, height/2, term.ColorBlack, term.ColorWhite)
        width, height = waitForResize(17, 17)
    }

    go updateSize()

    term.Clear(term.ColorWhite, term.ColorBlack)
    mainMenu()

    go keepConnection()
    
    for {
        play()
        win()

        if player == 0 {
            writeStr("Press enter to play again!", (width/2)-13, (height/2)+9, term.ColorBlack, term.ColorWhite )
            waitForEnter()
            reset()
        } else {
            waitForReplay()
        }
    }
}

func mainMenu() {
    for {
        choice := menu([][2]string{{"1.", " Create"}, {"2.", " Join"}}, true)
        if choice == 0 {
            var keys string
            for {
                term.Clear(term.ColorWhite, term.ColorBlack)
                writeStr(fmt.Sprint("Name: ", string(keys)), (width/2)-(len(fmt.Sprint("Name: ", string(keys)))/2), (height / 2), term.ColorBlack, term.ColorWhite)
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
            return
        } else if choice == 1 {
            var gameStrings [][2]string
            games := all()
            if len(games) < 1 {
                continue
            }
            for index, minimal := range games {
                gameStrings = append(gameStrings, [2]string{fmt.Sprintf("%v.", index), fmt.Sprintf(" %v          %v/2", minimal.Name, len(minimal.Players))})
            }
            index := menu(gameStrings, true)
            connect(games[index].ID)
            return
        }
    }
}

func play() {
    if len(TicTacToe.Players) < 2 {
        term.Clear(term.ColorWhite, term.ColorBlack)
        writeStr("Waiting for a player to join", (width/2)-14, height/2, term.ColorBlack, term.ColorWhite)
        waitForConnection()
    } else if player == TicTacToe.Player {
        row, column := boardSelect(boardChars(TicTacToe.Board), true)
        TicTacToe = update(row, column)
        boardSelect(boardChars(TicTacToe.Board), false)
    } else {
        boardSelect(boardChars(TicTacToe.Board), false)
    }

    for {
        waitForUpdate(TicTacToe.Turn)
        if TicTacToe.Winner != 0 {
            break
        }
        row, column := boardSelect(boardChars(TicTacToe.Board), true)
        TicTacToe = update(row, column)
        if TicTacToe.Winner != 0 {
            break
        }
        boardSelect(boardChars(TicTacToe.Board), false)
    }
}

func exitListener() {
    if err := keyboard.Open(); err != nil {
        panic(err)
    }
    defer keyboard.Close()
    for {
        if !listening {
            _, key, err := keyboard.GetKey()
            if key == keyboard.KeyEsc && err == nil {
                keyboard.Close()
                Exit()
            }
        }
        time.Sleep(100*time.Millisecond)
    }
} 

func Exit() {
    if connected {
        disconnect()
    }
    term.Close()
    pc, _, _, ok := runtime.Caller(1)
    details := runtime.FuncForPC(pc)
    if ok && details != nil {
        fmt.Printf("Exit called from %s\n", details.Name())
    }
    os.Exit(0)
}

func connection() {

}

func updateSize() {
    for {
        term.Flush()
        width, height = term.Size()
        time.Sleep(time.Second / 4)
    }
}

func menu(choices [][2]string, newLine bool) int {
    term.Clear(term.ColorWhite, term.ColorBlack)
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
            char = 0 - ((len(option[0]) + len(option[1]))/2)
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
            writeStr(option[0], (width/2)+(char/2), height/2, bg, fg)
            char += len(option[0])
            writeStr(option[1], (width/2)+(char/2), height/2, term.ColorBlack, term.ColorWhite)
            char += len(option[1])
            if index != len(choices)-1 {
                writeStr(" | ", (width/2)+(char/2), height/2, term.ColorBlack, term.ColorWhite)
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
                char = 0 - ((len(option[0]) + len(option[1]))/2)
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
                writeStr(option[0], (width/2)+(char/2), height/2, bg, fg)
                char += len(option[0])
                writeStr(option[1], (width/2)+(char/2), height/2, term.ColorBlack, term.ColorWhite)
                char += len(option[1])
                if index != len(choices)-1 {
                    writeStr(" | ", (width/2)+(char/2), height/2, term.ColorBlack, term.ColorWhite)
                    char += 3
                }
            }
        }
    }
}

func keepConnection() {
    for {
        if connected {
            url := fmt.Sprintf("https://TicTacToe.jettbergthold.repl.co/connection?gameid=%v&uid=%v", TicTacToe.ID, TicTacToe.Players[player])
            fakeJson, err := json.Marshal([]int{})
            if err != nil {
                panic(err)
            }
            resp, err := http.Post(url, "application/json", bytes.NewBuffer(fakeJson))
            if err != nil {
                panic(err)
            }
            resp.Body.Close()
        }
        time.Sleep(time.Second)
    }
}

func win() {
    var playerColor term.Attribute
    if TicTacToe.Winner == 1 {
        playerColor = term.ColorBlue
    } else {
        playerColor = term.ColorRed
    }
    writeHighlight(fmt.Sprintf("Player %v wins!", TicTacToe.Winner), [2]int{(width / 2) - 7, (height / 2) - 9}, [2][2]int{{7, 0}, {8, 0}}, [2]term.Attribute{term.ColorBlack, term.ColorWhite}, [2]term.Attribute{term.ColorBlack, playerColor})
    chars := boardChars(TicTacToe.Board)
    for row := 0; row < 3; row++ {
        for column := 0; column < 3; column++ {
            var tile string
            if row < 2 && column < 2 {
                tile = fmt.Sprintf(`     |
  %v  |
_____|`, chars[row][column])
            } else if row >= 2 && column >= 2 {
                tile = fmt.Sprintf(`     
  %v  
     `, chars[row][column])
            } else if row >= 2 {
                tile = fmt.Sprintf(`     |
  %v  |
     |`, chars[row][column])
            } else if column >= 2 {
                tile = fmt.Sprintf(`     
  %v  
_____`, chars[row][column])
            }
            var highColor [2]term.Attribute
            if winContains(TicTacToe.WinTiles, [2]int{row, column}) {
                highColor = [2]term.Attribute{term.ColorWhite, term.ColorBlack}
            } else {
                highColor = [2]term.Attribute{term.ColorBlack, term.ColorWhite}
            }
            writeHighlight(tile, [2]int{((width / 2) - 8) + (6 * column), ((height / 2) - 8) + (3 * row)}, [2][2]int{{2, 1}, {2, 1}}, [2]term.Attribute{term.ColorBlack, term.ColorWhite}, highColor)
        }
    }
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
    return newGame
}

func create(name string) {
    TicTacToe = gameFromPost(fmt.Sprint("https://TicTacToe.jettbergthold.repl.co/create?name=", urlLib.QueryEscape(name)))
    connected = true
    player = 0
}

func connect(id int) {
    TicTacToe = gameFromPost(fmt.Sprint("https://TicTacToe.jettbergthold.repl.co/connect?gameid=", id))
    connected = true
    player = 1
}

func update(row int, column int) game {
    updated := gameFromPost(fmt.Sprint("https://TicTacToe.jettbergthold.repl.co/update?gameid=", TicTacToe.ID, "&uid=", TicTacToe.Players[player], "&row=", row, "&column=", column))
    return updated
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

func disconnect() {
    gameFromPost(fmt.Sprint("https://TicTacToe.jettbergthold.repl.co/disconnect?gameid=", TicTacToe.ID, "&uid=", TicTacToe.Players[player]))
    connected = false
}

func reset() {
    TicTacToe = gameFromPost(fmt.Sprint("https://TicTacToe.jettbergthold.repl.co/reset?gameid=", TicTacToe.ID, "&uid=", TicTacToe.Players[player]))
}

func waitForUpdate(turn int) {
    //go exitListener()
    for {
        var updated game
        disconnectCheck(TicTacToe.Players)
        resp, err := http.Get(fmt.Sprintf("https://TicTacToe.jettbergthold.repl.co/update?gameid=%v&uid=%v", TicTacToe.ID, TicTacToe.Players[player]))
        if err != nil {
            panic(err)
        }
        defer resp.Body.Close()
        json.NewDecoder(resp.Body).Decode(&updated)
        if updated.Turn > turn {
            TicTacToe = updated
            disconnectCheck(TicTacToe.Players)
            return
        }
        time.Sleep(time.Second >> 2)
    }
}

func disconnectCheck(players []string) {
    if len(players) < 2 {
        player = 0
        term.Clear(term.ColorWhite, term.ColorBlack)
        writeStr("Opponent disconnected!\nWaiting for connection", (width/2)-11, (height/2)-1, term.ColorBlack, term.ColorWhite)
        waitForConnection()
    }
}

func waitForConnection() {
    //go exitListener()
    for {
        resp, err := http.Get(fmt.Sprintf("https://TicTacToe.jettbergthold.repl.co/update?gameid=%v&uid=%v", TicTacToe.ID, TicTacToe.Players[player]))
        if err != nil {
            panic(err)
        }
        defer resp.Body.Close()
        var updated minimalGame
        json.NewDecoder(resp.Body).Decode(&updated)

        if len(updated.Players) > 1 {
            return
        }

        time.Sleep(time.Second >> 2)
    }
}

func waitForReplay() {
    //go exitListener()
    for {
        resp, err := http.Get(fmt.Sprintf("https://TicTacToe.jettbergthold.repl.co/update?gameid=%v&uid=%v", TicTacToe.ID, TicTacToe.Players[player]))
        if err != nil {
            panic(err)
        }
        defer resp.Body.Close()
        var updated game
        json.NewDecoder(resp.Body).Decode(&updated)

        if updated.Turn == 0 {
            return
        }
        disconnectCheck(updated.Players)

        time.Sleep(time.Second >> 2)
    }
}

func waitForEnter() {
    listening = true
    defer func() {listening = false}()
    for {
        ev := term.PollEvent()
        if ev.Type == term.EventKey {
            if ev.Key == term.KeyEnter {
                return
            } else if ev.Key == term.KeyEsc {
                Exit()
            }
        }
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
    //go exitListener()
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
    listening = true
    defer func() {listening = false}()
    for {
        ev := term.PollEvent()
        switch ev.Type {
        case term.EventKey:
            switch ev.Key {
            case term.KeyArrowUp:
                term.Sync()
                return 1
            case term.KeyArrowDown:
                term.Sync()
                return 3
            case term.KeyArrowLeft:
                term.Sync()
                return 2
            case term.KeyArrowRight:
                term.Sync()
                return 4
            case term.KeyEnter:
                return 0
            case term.KeyEsc:
                Exit()
            }
        case term.EventError:
            panic(ev.Err)
        }
    }
}

func keyListener() (term.Key, rune) {
    listening = true
    defer func() {listening = false}()
    for {
        ev := term.PollEvent()
        if ev.Type == term.EventKey && ev.Key != term.KeyEsc {
            return ev.Key, ev.Ch
        } else if ev.Type == term.EventKey && ev.Key == term.KeyEsc {
            Exit()
        }
    }
}

func clear() {
    cmd := exec.Command("cmd", "/c", "cls")
    cmd.Stdout = os.Stdout
    cmd.Run()
}

func popString(slice string) string {
    return slice[:len(slice)-1]
}

func winContains(tiles [3][2]int, pos [2]int) bool {
    for _, tile := range tiles {
        if pos == tile {
            return true
        }
    }
    return false
}

func boardSelect(chars [3][3]string, selecting bool) (int, int) {
    term.Clear(term.ColorWhite, term.ColorBlack)
    selection := [2]int{0, 0}
    board := fmt.Sprintf(`     |     |     
  %v  |  %v  |  %v  
_____|_____|_____
     |     |     
  %v  |  %v  |  %v   
_____|_____|_____
     |     |     
  %v  |  %v  |  %v  
     |     |     `, chars[0][0], chars[0][1], chars[0][2], chars[1][0], chars[1][1], chars[1][2], chars[2][0], chars[2][1], chars[2][2])
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
                if chars[selection[1]][selection[0]] == "-" {
                    return selection[1], selection[0]
                }
            }
            writeHighlight(board, [2]int{(width / 2) - 8, (height / 2) - 8}, [2][2]int{{(selection[0] * 6) + 2, (selection[1] * 3) + 1}, {(selection[0] * 6) + 2, (selection[1] * 3) + 1}}, [2]term.Attribute{term.ColorBlack, term.ColorWhite}, [2]term.Attribute{term.ColorWhite, term.ColorBlack})
        }
    } else {
        writeStr(board, (width/2)-8, (height/2)-8, term.ColorBlack, term.ColorWhite)
        return -1, -1
    }
}
