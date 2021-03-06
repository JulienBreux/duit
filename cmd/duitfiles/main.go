package main

import (
	"bufio"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

type column struct {
	name  string
	names []string
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func open(path string) {
	log.Printf("open %s\n", path)
	// xxx should be per platform, might want to try plumbing first.
	err := exec.Command("open", path).Run()
	if err != nil {
		log.Printf("open: %s\n", err)
	}
}

func favoritesPath() string {
	return os.Getenv("HOME") + "/lib/duit/files/favorites"
}

func loadFavorites() ([]*duit.ListValue, error) {
	home := os.Getenv("HOME") + "/"
	l := []*duit.ListValue{
		{Text: "home", Value: home, Selected: true},
		{Text: "/", Value: "/"},
	}

	f, err := os.Open(favoritesPath())
	if os.IsNotExist(err) {
		return l, nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		name := scanner.Text()
		l = append(l, &duit.ListValue{Text: path.Base(name), Value: name})
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func saveFavorites(l []*duit.ListValue) (err error) {
	favPath := favoritesPath()
	os.MkdirAll(path.Dir(favPath), 0777)
	f, err := os.Create(favPath)
	if err != nil {
		return err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()
	for _, lv := range l[2:] {
		_, err = fmt.Fprintln(f, lv.Value.(string))
		if err != nil {
			return
		}
	}
	err = f.Close()
	f = nil
	return
}

func listDir(path string) []string {
	files, err := ioutil.ReadDir(path)
	check(err, "readdir")
	names := make([]string, len(files))
	for i, fi := range files {
		names[i] = fi.Name()
		if fi.IsDir() {
			names[i] += "/"
		}
	}
	return names
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("files: ")

	dui, err := duit.NewDUI("files", nil)
	check(err, "new dui")

	// layout: favorites on the left, fixed size. remainder on the right contains one or more listboxes.
	// favorites is populated with some names that point to dirs. clicking makes the favorite active, and focuses on first column.
	// typing then filters only the matching elements.  we just show text. names ending in "/" are directories.
	// hitting tab on a directory opens that dir, and moves focus there.
	// hitting "enter" on a file causes it to be plumbed (opened).

	var (
		selectName     func(int, string)
		composePath    func(int, string) string
		columnsUI      *duit.Split
		favoritesUI    *duit.List
		favoriteToggle *duit.Button
		activeFavorite *duit.ListValue
		makeColumnUI   func(colIndex int, c column) duit.UI
	)

	favorites, err := loadFavorites()
	check(err, "loading favorites")

	columns := []column{
		{names: listDir(favorites[0].Value.(string))},
	}
	pathLabel := &duit.Label{Text: favorites[0].Value.(string)}

	favoritesUI = &duit.List{
		Values: favorites,
		Changed: func(index int) (e duit.Event) {
			activeFavorite = favoritesUI.Values[index]
			activeFavorite.Selected = true
			path := activeFavorite.Value.(string)
			pathLabel.Text = path
			favoriteToggle.Text = "-"
			columns = []column{
				{name: "", names: listDir(path)},
			}
			columnsUI.Kids = duit.NewKids(makeColumnUI(0, columns[0]))
			e.NeedLayout = true // xxx probably propagate to top?
			return
		},
	}
	activeFavorite = favoritesUI.Values[0]

	findFavorite := func(path string) *duit.ListValue {
		for _, lv := range favoritesUI.Values {
			if lv.Value.(string) == path {
				return lv
			}
		}
		return nil
	}

	favoriteToggle = &duit.Button{
		Text: "-",
		Click: func() (e duit.Event) {
			log.Printf("toggle favorite\n")
			for _, lv := range favoritesUI.Values {
				lv.Selected = false
			}
			lv := findFavorite(pathLabel.Text)
			if lv == favoritesUI.Values[0] {
				return
			}
			if lv == nil {
				lv = &duit.ListValue{
					Text:     path.Base(pathLabel.Text),
					Value:    pathLabel.Text,
					Selected: true,
				}
				favoritesUI.Values = append(favoritesUI.Values, lv)
			} else {
				var nl []*duit.ListValue
				for _, lv := range favoritesUI.Values {
					if lv.Value.(string) != pathLabel.Text {
						nl = append(nl, lv)
					}
				}
				favoritesUI.Values = nl
			}
			err := saveFavorites(favoritesUI.Values)
			check(err, "saving favorites")
			e.NeedLayout = true // xxx probably propagate to top
			return
		},
	}

	makeColumnUI = func(colIndex int, c column) duit.UI {
		l := make([]*duit.ListValue, len(c.names))
		for i, name := range c.names {
			l[i] = &duit.ListValue{Text: name, Value: name}
		}
		var list *duit.List
		list = &duit.List{
			Values: l,
			Changed: func(index int) (e duit.Event) {
				if list.Values[index].Selected {
					selectName(colIndex, list.Values[index].Value.(string))
					e.NeedLayout = true // xxx propagate to top?
				} else {
					selectName(colIndex, "")
				}
				return
			},
			Click: func(index int, m draw.Mouse) (e duit.Event) {
				if m.Buttons != 1<<2 {
					return
				}
				path := composePath(colIndex, list.Values[index].Value.(string))
				open(path)
				e.Consumed = true
				return
			},
			Keys: func(k rune, m draw.Mouse) (e duit.Event) {
				log.Printf("list.keys, k %x %c %v\n", k, k, k)
				switch k {
				case '\n':
					sel := list.Selected()
					if len(sel) != 1 {
						return
					}
					index := sel[0]
					e.Consumed = true
					path := composePath(colIndex, list.Values[index].Value.(string))
					open(path)
				case draw.KeyLeft:
					e.Consumed = true
					if colIndex > 0 {
						selectName(colIndex-1, "")
						e.NeedLayout = true
					} else {
						selectName(colIndex, "")
						e.NeedDraw = true
					}
				case draw.KeyRight:
					sel := list.Selected()
					if len(sel) != 1 {
						return
					}
					index := sel[0]
					elem := list.Values[index].Value.(string)
					log.Printf("arrow right, index %d, elem %s\n", index, elem)
					if strings.HasSuffix(elem, "/") {
						e.Consumed = true
						selectName(colIndex, elem)
						if len(columns[colIndex+1].names) > 0 {
							log.Printf("selecting next first in new column\n")
							selectName(colIndex+1, columns[colIndex+1].names[0])
						}
						dui.Render()
						newList := columnsUI.Kids[len(columnsUI.Kids)-1].UI.(*duit.Box).Kids[1].UI.(*duit.Scroll).Kid.UI
						dui.Focus(newList)
						e.NeedLayout = true // xxx propagate?
					}
				}
				return
			},
		}
		return &duit.Box{
			Padding: duit.SpaceXY(6, 4),
			Margin:  image.Pt(6, 4),
			Kids: duit.NewKids(
				&duit.Field{
					Changed: func(newValue string) (e duit.Event) {
						nl := []*duit.ListValue{}
						exactMatch := false
						for _, name := range c.names {
							exactMatch = exactMatch || name == newValue
							if strings.Contains(name, newValue) {
								nl = append(nl, &duit.ListValue{Text: name, Value: name})
							}
						}
						if exactMatch {
							selectName(colIndex, newValue)
							dui.Render()
							field := columnsUI.Kids[len(columnsUI.Kids)-1].UI.(*duit.Box).Kids[0].UI.(*duit.Field)
							dui.Focus(field)
						}
						list.Values = nl
						e.NeedLayout = true // xxx propagate?
						return
					},
				},
				duit.NewScroll(list),
			),
		}
	}

	columnsUI = &duit.Split{
		Gutter: 1,
		Split: func(width int) []int {
			widths := make([]int, len(columns))
			col := width / len(widths)
			for i := range widths {
				widths[i] = col
			}
			widths[len(widths)-1] = width - col*(len(widths)-1)
			// xxx should layout more dynamically.  taking max of what is needed and what is available. and giving more to column of focus.  might need horizontal scroll too.
			return widths
		},
		Kids: duit.NewKids(makeColumnUI(0, columns[0])),
	}

	composePath = func(col int, name string) string {
		path := activeFavorite.Value.(string)
		for _, column := range columns[:col] {
			path += column.name
		}
		path += name
		return path
	}

	selectName = func(col int, name string) {
		log.Printf("selectName col %d, name %s\n", col, name)
		path := activeFavorite.Value.(string)
		columns = columns[:col+1]
		columns[col].name = name
		columnsUI.Kids = columnsUI.Kids[:col+1]
		for _, column := range columns {
			path += column.name
		}
		pathLabel.Text = path
		if findFavorite(path) == nil {
			favoriteToggle.Text = "+"
		} else {
			favoriteToggle.Text = "-"
		}
		if !strings.HasSuffix(path, "/") {
			// not a dir, nothing to do for file selection
			return
		}
		names := listDir(path)
		if name == "" {
			// no new column to show
			return
		}
		newCol := column{name: name, names: names}
		columns = append(columns, newCol)
		columnsUI.Kids = append(columnsUI.Kids, &duit.Kid{UI: makeColumnUI(len(columns)-1, newCol)})
	}

	dui.Top.UI = &duit.Box{
		Padding: duit.SpaceXY(6, 4),
		Margin:  image.Pt(6, 4),
		Valign:  duit.ValignMiddle,
		Kids: duit.NewKids(
			favoriteToggle,
			pathLabel,
			&duit.Split{
				Gutter: 1,
				Split: func(width int) []int {
					return []int{dui.Scale(200), width - dui.Scale(200)}
				},
				Kids: duit.NewKids(favoritesUI, columnsUI),
			},
		),
	}
	dui.Render()
	dui.Focus(columnsUI.Kids[0].UI.(*duit.Box).Kids[0].UI.(*duit.Field))

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)

		case err := <-dui.Error:
			check(err, "dui")
			return
		}
	}
}
