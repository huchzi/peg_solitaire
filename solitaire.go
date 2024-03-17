package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

type JumpType string

const (
	horizontal JumpType = "H"
	vertical   JumpType = "V"
)

type ButtonEnabled template.HTMLAttr

const (
	enabled   ButtonEnabled = ""
	disabled  ButtonEnabled = "disabled"
	invisible ButtonEnabled = "hidden"
)

const (
	leftArrow    string = string(rune(8592))
	rightArrow   string = string(rune(8594))
	upArrow      string = string(rune(8593))
	downArrow    string = string(rune(8595))
	leftAndRight string = string(rune(8596))
	upAndDown    string = string(rune(8597))
)

type Field struct {
	FieldID             string
	Class               template.CSS
	Clickable           ButtonEnabled
	HorizontalNeighbors [2]*Field
	VerticalNeighbors   [2]*Field
	PossibleJumpType    JumpType
	Arrow               string
}

var NoStone = Field{
	Class:     "noStone",
	Clickable: ""}

var Stone = Field{
	Class:     "stone",
	Clickable: "",
	Arrow:     leftArrow}

var NonExistant = Field{
	Class:     "nothing",
	Clickable: "disabled"}

var templates = template.Must(template.ParseFiles("playingField.html"))

var rows = []string{"1", "2", "3", "4", "5", "6", "7"}
var cols = []string{"A", "B", "C", "D", "E", "F", "G"}

type GameState struct {
	PlayingField  [][]Field
	AllFields     map[string]*Field
	Choice        ButtonEnabled
	Button1       string
	Button2       string
	Direction     JumpType
	SelectedField *Field
	History       []string
	nMoves        int
}

var MainState = GameState{Choice: invisible, Direction: horizontal, Button1: "O", Button2: "O"}

// Init sets up the playing field and the game state in a GameState struct.8
func Init() {
	MainState.AllFields = make(map[string]*Field, 0)
	MainState.PlayingField = make([][]Field, 7)
	for i := range MainState.PlayingField {
		if i > 1 && i < 5 {
			MainState.PlayingField[i] = []Field{Stone, Stone, Stone, Stone, Stone, Stone, Stone}
		} else {
			MainState.PlayingField[i] = []Field{NonExistant, NonExistant, Stone, Stone, Stone, NonExistant, NonExistant}
		}
	}
	for i := range MainState.PlayingField {
		for j := range MainState.PlayingField[i] {
			fieldID := cols[j] + rows[i]
			MainState.PlayingField[i][j].FieldID = fieldID
			MainState.AllFields[fieldID] = &MainState.PlayingField[i][j]
			if j > 0 && j < 6 {
				MainState.PlayingField[i][j].HorizontalNeighbors[0] = &MainState.PlayingField[i][j-1]
				MainState.PlayingField[i][j].HorizontalNeighbors[1] = &MainState.PlayingField[i][j+1]
			}
			if i > 0 && i < 6 {
				MainState.PlayingField[i][j].VerticalNeighbors[0] = &MainState.PlayingField[i-1][j]
				MainState.PlayingField[i][j].VerticalNeighbors[1] = &MainState.PlayingField[i+1][j]
			}
		}
	}
	MainState.AllFields["D4"].Toggle()
	MainState.UpdatePossibleMoves()
	MainState.History = make([]string, 0)
	MainState.nMoves = 0
}

// UpdatePossibleMoves identifies possible moves in the playing field in the GameState "st".
func (st *GameState) UpdatePossibleMoves() {
	for _, field := range st.AllFields {
		field.Clickable = disabled
		field.Arrow = ""
	}
	for _, field := range st.AllFields {
		field.MovePossible(horizontal)
		field.MovePossible(vertical)
	}
}

// MovePossible checks for possible moves in "f" and changes the state of "f" when a move is possible.
func (f *Field) MovePossible(jt JumpType) {
	var n1 *Field
	var n2 *Field
	var a1, a2 string
	if jt == horizontal {
		n1 = f.HorizontalNeighbors[0]
		n2 = f.HorizontalNeighbors[1]
		a1, a2 = rightArrow, leftArrow
	} else {
		n1 = f.VerticalNeighbors[0]
		n2 = f.VerticalNeighbors[1]
		a1, a2 = downArrow, upArrow
	}
	switch {
	case f.Class == "noStone":
		f.Clickable = "disabled"
	case n1 == nil || n2 == nil:
	case n1.Class == "nothing" || n2.Class == "nothing":
	case n1.Class == "stone":
		if n2.Class == "noStone" {
			f.Clickable = ""
			f.PossibleJumpType = jt
			f.Arrow = f.Arrow + a1
		}
	case n1.Class == "noStone":
		if n2.Class == "stone" {
			f.Clickable = ""
			f.PossibleJumpType = jt
			f.Arrow = f.Arrow + a2
		}
	}
}

// Toogle changes the state of a field from "Stone" to "NoStone" or vice versa.
func (f *Field) Toggle() {
	var fNew Field
	if f.Class == "noStone" {
		fNew = Stone
	} else {
		fNew = NoStone
	}
	fNew.FieldID = f.FieldID
	fNew.HorizontalNeighbors = f.HorizontalNeighbors
	fNew.VerticalNeighbors = f.VerticalNeighbors
	*f = fNew
}

// Jump performs a jump by modifying the field and updating the game state.
func (f *Field) Jump(jt JumpType) {
	if jt == horizontal {
		f.HorizontalNeighbors[0].Toggle()
		f.HorizontalNeighbors[1].Toggle()
	} else {
		f.VerticalNeighbors[0].Toggle()
		f.VerticalNeighbors[1].Toggle()
	}
	f.Toggle()
	MainState.UpdatePossibleMoves()
	MainState.nMoves += 1
	thisMove := fmt.Sprintf("(%d) %s", MainState.nMoves, f.FieldID+string(jt))
	MainState.History = append(MainState.History, thisMove)
}

// GoToHistory reverts the gamestate to an earlier state defined by the last move.
func GoToHistory(goTo string) {
	var backupHistory []string = make([]string, len(MainState.History))
	copy(backupHistory, MainState.History)
	Init()
	for _, mv := range backupHistory {
		if strings.HasSuffix(mv, "H") {
			field := strings.Split(mv, " ")[1]
			field, _ = strings.CutSuffix(field, "H")
			MainState.AllFields[field].Jump(horizontal)
		} else {
			field := strings.Split(mv, " ")[1]
			field, _ = strings.CutSuffix(field, "V")
			MainState.AllFields[field].Jump(vertical)
		}
		if mv == goTo {
			break
		}
	}
	MainState.History = backupHistory
}

// Update listens for user input and reacts according to the value of the input form field "field".
func Update(w http.ResponseWriter, r *http.Request) {
	sel := r.FormValue("field")
	switch {
	case sel == "Reset":
		Init()
	case sel == "Save History":
		history, err := json.Marshal(MainState.History)
		if err != nil {
			fmt.Println(err)
		}
		err = os.WriteFile("history.solitaire", history, os.ModeAppend)
		if err != nil {
			fmt.Println(err)
		}
	case sel == "Load History":
		jsn, err := os.ReadFile("history.solitaire")
		if err != nil {
			panic("Unable to read file.")
		}
		err = json.Unmarshal(jsn, &(MainState.History))
		if err != nil {
			panic("Cannot marshall history.")
		}
	case sel == "Horizontal":
		MainState.Choice = invisible
		MainState.SelectedField.Jump(horizontal)
	case sel == "Vertical":
		MainState.Choice = invisible
		MainState.SelectedField.Jump(vertical)
	case sel == "Undo":
		switch len(MainState.History) {
		case 0:
		case 1:
			Init()
		default:
			GoToHistory(MainState.History[len(MainState.History)-2])
			MainState.History = MainState.History[0 : len(MainState.History)-1]
		}
	case strings.HasSuffix(sel, "H") || strings.HasSuffix(sel, "V"):
		GoToHistory(sel)
	case sel == "":
	default:
		thisField := MainState.AllFields[r.FormValue("field")]
		if thisField == nil {
			break
		}
		if len(thisField.Arrow) == 6 {
			MainState.Choice = enabled
			MainState.Button1 = strings.Split(thisField.Arrow, "")[0]
			MainState.Button2 = strings.Split(thisField.Arrow, "")[1]
			MainState.SelectedField = thisField
			thisField.Class = "selector"
			for _, field := range MainState.AllFields {
				field.Clickable = "disabled"
				field.Arrow = ""
			}
		} else {
			thisField.Jump(thisField.PossibleJumpType)
		}
	}
	templates.ExecuteTemplate(w, "playingField.html", MainState)
}

func main() {

	Init()

	http.HandleFunc("/", Update)
	log.Fatal(http.ListenAndServe(":8080", nil))

}
