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

var GameState = struct {
	PlayingField  [][]Field
	AllFields     map[string]*Field
	Choice        ButtonEnabled
	Button1       string
	Button2       string
	Direction     JumpType
	SelectedField *Field
	History       []string
	nMoves        int
}{Choice: invisible, Direction: horizontal, Button1: "O", Button2: "O"}

func Init() {
	GameState.AllFields = make(map[string]*Field, 0)
	GameState.PlayingField = make([][]Field, 7)
	for i := range GameState.PlayingField {
		if i > 1 && i < 5 {
			GameState.PlayingField[i] = []Field{Stone, Stone, Stone, Stone, Stone, Stone, Stone}
		} else {
			GameState.PlayingField[i] = []Field{NonExistant, NonExistant, Stone, Stone, Stone, NonExistant, NonExistant}
		}
	}
	for i := range GameState.PlayingField {
		for j := range GameState.PlayingField[i] {
			fieldID := cols[j] + rows[i]
			GameState.PlayingField[i][j].FieldID = fieldID
			GameState.AllFields[fieldID] = &GameState.PlayingField[i][j]
			if j > 0 && j < 6 {
				GameState.PlayingField[i][j].HorizontalNeighbors[0] = &GameState.PlayingField[i][j-1]
				GameState.PlayingField[i][j].HorizontalNeighbors[1] = &GameState.PlayingField[i][j+1]
			}
			if i > 0 && i < 6 {
				GameState.PlayingField[i][j].VerticalNeighbors[0] = &GameState.PlayingField[i-1][j]
				GameState.PlayingField[i][j].VerticalNeighbors[1] = &GameState.PlayingField[i+1][j]
			}
		}
	}
	GameState.AllFields["D4"].Toggle()
	UpdatePossibleMoves()
	GameState.History = make([]string, 0)
	GameState.nMoves = 0
}

func UpdatePossibleMoves() {
	for _, field := range GameState.AllFields {
		field.Clickable = disabled
		field.Arrow = ""
	}
	for _, field := range GameState.AllFields {
		field.MovePossible(horizontal)
		field.MovePossible(vertical)
	}
}

func GoToHistory(goTo string) {
	var backupHistory []string = make([]string, len(GameState.History))
	copy(backupHistory, GameState.History)
	Init()
	for _, mv := range backupHistory {
		if strings.HasSuffix(mv, "H") {
			field := strings.Split(mv, " ")[1]
			field, _ = strings.CutSuffix(field, "H")
			GameState.AllFields[field].Jump(horizontal)
		} else {
			field := strings.Split(mv, " ")[1]
			field, _ = strings.CutSuffix(field, "V")
			GameState.AllFields[field].Jump(vertical)
		}
		if mv == goTo {
			break
		}
	}
	GameState.History = backupHistory
}

func drawPlayingField(w http.ResponseWriter, r *http.Request) {
	sel := r.FormValue("field")
	switch {
	case sel == "Reset":
		Init()
	case sel == "Save History":
		history, _ := json.Marshal(GameState.History)
		os.WriteFile("history.solitaire", history, os.ModeAppend)
	case sel == "Load History":
		jsn, err := os.ReadFile("history.solitaire")
		if err != nil {
			panic("Unable to read file.")
		}
		err = json.Unmarshal(jsn, &(GameState.History))
		if err != nil {
			panic("Cannot marshall history.")
		}
	case sel == "Horizontal":
		GameState.Choice = invisible
		GameState.SelectedField.Jump(horizontal)
	case sel == "Vertical":
		GameState.Choice = invisible
		GameState.SelectedField.Jump(vertical)
	case sel == "Undo":
		switch len(GameState.History) {
		case 0:
		case 1:
			Init()
		default:
			GoToHistory(GameState.History[len(GameState.History)-2])
			GameState.History = GameState.History[0 : len(GameState.History)-1]
		}
	case strings.HasSuffix(sel, "H") || strings.HasSuffix(sel, "V"):
		GoToHistory(sel)
	case sel == "":
	default:
		thisField := GameState.AllFields[r.FormValue("field")]
		if thisField == nil {
			break
		}
		if len(thisField.Arrow) == 6 {
			GameState.Choice = enabled
			GameState.Button1 = strings.Split(thisField.Arrow, "")[0]
			GameState.Button2 = strings.Split(thisField.Arrow, "")[1]
			GameState.SelectedField = thisField
			thisField.Class = "selector"
			for _, field := range GameState.AllFields {
				field.Clickable = "disabled"
				field.Arrow = ""
			}
		} else {
			thisField.Jump(thisField.PossibleJumpType)
		}
	}
	templates.ExecuteTemplate(w, "playingField.html", GameState)
}

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

func (f *Field) Jump(jt JumpType) {
	if jt == horizontal {
		f.HorizontalNeighbors[0].Toggle()
		f.HorizontalNeighbors[1].Toggle()
	} else {
		f.VerticalNeighbors[0].Toggle()
		f.VerticalNeighbors[1].Toggle()
	}
	f.Toggle()
	UpdatePossibleMoves()
	GameState.nMoves += 1
	thisMove := fmt.Sprintf("(%d) %s", GameState.nMoves, f.FieldID+string(jt))
	GameState.History = append(GameState.History, thisMove)
}

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

func main() {

	Init()

	http.HandleFunc("/", drawPlayingField)
	log.Fatal(http.ListenAndServe(":8080", nil))

}
