package main

import (
    "bufio"
    "fmt"
    "os"
    "regexp"
    "strings"
    "unicode/utf8"

    "github.com/gdamore/tcell/v2"
    "golang.org/x/text/encoding/charmap"
    "golang.org/x/text/transform"
)

type HighlightRule struct {
    pattern *regexp.Regexp
    style   tcell.Style
}

type LanguageRules struct {
    name  string
    rules []HighlightRule
}

type Buffer struct {
    lines     []string
    filename  string
    cursorX   int
    cursorY   int
    offsetX   int
    offsetY   int
    saved     bool
}

type Editor struct {
    screen      tcell.Screen
    buffers     []*Buffer
    activeBuf   int
    status      string
    quit        bool
    highlight   bool
    languages   map[string]LanguageRules
    theme       map[string]tcell.Style
}

func NewEditor() *Editor {
    // Color de fondo #1e2027 (RGB: 30, 32, 39)
    bgColor := tcell.NewRGBColor(30, 32, 39)
    defaultStyle := tcell.StyleDefault.Background(bgColor).Foreground(tcell.ColorWhite)

    editor := &Editor{
        buffers:   []*Buffer{{lines: []string{""}, saved: false}},
        status:    "Welcome to GoEditor | Ctrl-Q: Quit | Ctrl-S: Save | F1: Toggle Highlight | Ctrl-T: New Tab | Ctrl-W: Close Tab | Ctrl-PgUp/PgDn: Switch Tabs",
        highlight: true,
        languages: make(map[string]LanguageRules),
    }

    // Configurar tema de colores
    editor.theme = make(map[string]tcell.Style)
    editor.theme["background"] = defaultStyle
    editor.theme["text"] = defaultStyle
    editor.theme["keyword"] = defaultStyle.Foreground(tcell.NewRGBColor(197, 134, 192)) // #c586c0
    editor.theme["type"] = defaultStyle.Foreground(tcell.NewRGBColor(78, 201, 176))     // #4ec9b0
    editor.theme["string"] = defaultStyle.Foreground(tcell.NewRGBColor(206, 145, 120))  // #ce9178
    editor.theme["comment"] = defaultStyle.Foreground(tcell.NewRGBColor(106, 153, 85))  // #6a9955
    editor.theme["number"] = defaultStyle.Foreground(tcell.NewRGBColor(181, 206, 168))  // #b5cea8
    editor.theme["function"] = defaultStyle.Foreground(tcell.NewRGBColor(220, 220, 170)) // #dcdcaa
    editor.theme["preprocessor"] = defaultStyle.Foreground(tcell.NewRGBColor(155, 155, 255)) // #9b9bff
    editor.theme["division"] = defaultStyle.Foreground(tcell.NewRGBColor(255, 215, 0))      // #ffd700 (gold)
    editor.theme["tab_active"] = tcell.StyleDefault.
        Background(tcell.NewRGBColor(70, 70, 90)).
        Foreground(tcell.ColorWhite).
        Bold(true)
    editor.theme["tab_inactive"] = tcell.StyleDefault.
        Background(tcell.NewRGBColor(50, 50, 60)).
        Foreground(tcell.ColorSilver)
    editor.theme["status"] = tcell.StyleDefault.
        Background(tcell.NewRGBColor(37, 37, 43)). // #25252b
        Foreground(tcell.NewRGBColor(204, 204, 204)).
        Bold(true)

    // Configurar reglas de resaltado para cada lenguaje
    editor.setupGoHighlightRules()
    editor.setupCHighlightRules()
    editor.setupCOBOLHighlightRules()

    return editor
}

func (e *Editor) setupGoHighlightRules() {
    keywords := `break|default|func|interface|select|case|defer|go|map|struct` +
        `|chan|else|goto|package|switch|const|fallthrough|if|range|type` +
        `|continue|for|import|return|var`

    types := `int|int8|int16|int32|int64|uint|uint8|uint16|uint32|uint64` +
        `|float32|float64|complex64|complex128|byte|rune|string|bool|error`

    e.languages["go"] = LanguageRules{
        name: "Go",
        rules: []HighlightRule{
            {regexp.MustCompile(`//.*$`), e.theme["comment"]},
            {regexp.MustCompile(`/\*.*?\*/`), e.theme["comment"]},
            {regexp.MustCompile(`\b(` + keywords + `)\b`), e.theme["keyword"]},
            {regexp.MustCompile(`\b(` + types + `)\b`), e.theme["type"]},
            {regexp.MustCompile(`\b\d+(\.\d+)?\b`), e.theme["number"]},
            {regexp.MustCompile(`".*?"`), e.theme["string"]},
            {regexp.MustCompile(`'.*?'`), e.theme["string"]},
            {regexp.MustCompile(`\b\w+\(`), e.theme["function"]},
        },
    }
}

func (e *Editor) setupCHighlightRules() {
    keywords := `auto|break|case|char|const|continue|default|do|double|else` +
        `|enum|extern|float|for|goto|if|int|long|register|return` +
        `|short|signed|sizeof|static|struct|switch|typedef|union` +
        `|unsigned|void|volatile|while`

    types := `int|char|float|double|short|long|void|size_t|ssize_t`

    e.languages["c"] = LanguageRules{
        name: "C",
        rules: []HighlightRule{
            {regexp.MustCompile(`//.*$`), e.theme["comment"]},
            {regexp.MustCompile(`/\*.*?\*/`), e.theme["comment"]},
            {regexp.MustCompile(`^\s*#\s*\w+`), e.theme["preprocessor"]},
            {regexp.MustCompile(`\b(` + keywords + `)\b`), e.theme["keyword"]},
            {regexp.MustCompile(`\b(` + types + `)\b`), e.theme["type"]},
            {regexp.MustCompile(`\b\d+(\.\d+)?([eE][+-]?\d+)?[fFlL]?\b`), e.theme["number"]},
            {regexp.MustCompile(`\b0[xX][0-9a-fA-F]+\b`), e.theme["number"]},
            {regexp.MustCompile(`".*?"`), e.theme["string"]},
            {regexp.MustCompile(`'.*?'`), e.theme["string"]},
            {regexp.MustCompile(`\b\w+\(`), e.theme["function"]},
        },
    }
}

func (e *Editor) setupCOBOLHighlightRules() {
    divisions := `IDENTIFICATION|ENVIRONMENT|DATA|PROCEDURE`
    keywords := `ACCEPT|ADD|ALLOCATE|CALL|CANCEL|CLOSE|COMPUTE|CONTINUE|COPY` +
        `|DELETE|DISPLAY|DIVIDE|ELSE|END|EVALUATE|EXIT|FREE|GO|GOBACK` +
        `|IF|INITIALIZE|INSPECT|INVOKE|MOVE|MULTIPLY|OPEN|PERFORM|READ` +
        `|RELEASE|RETURN|REWRITE|SEARCH|SET|SORT|START|STOP|STRING` +
        `|SUBTRACT|UNSTRING|WRITE|WHEN|THROUGH|TIMES|UNTIL|VARYING`

    rules := []struct {
        pattern string
        style   tcell.Style
    }{
        {`^\*.*$`, e.theme["comment"]},
        {`^......\*.*$`, e.theme["comment"]},
        {`^......\/\*.*$`, e.theme["comment"]},
        {`^\s*(` + divisions + `)\s+DIVISION\.`, e.theme["division"]},
        {`^\s*(` + divisions + `)\s+DIVISION\s*\.`, e.theme["division"]},
        {`^\s*[A-Z0-9-]+\s+SECTION\.`, e.theme["division"]},
        {`\b(` + keywords + `)\b`, e.theme["keyword"]},
        {`^\s*(0[1-9]|[1-4][0-9]|77|88)\s+`, e.theme["number"]},
        {`\b\d+\b`, e.theme["number"]},
        {`"[^"]*"`, e.theme["string"]},
        {`'[^']*'`, e.theme["string"]},
        {`^\s*[A-Z0-9-]+\s*\.$`, e.theme["function"]},
    }

    var compiledRules []HighlightRule
    for _, rule := range rules {
        re, err := regexp.Compile(`(?i)` + rule.pattern)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error compiling COBOL regex %s: %v\n", rule.pattern, err)
            continue
        }
        compiledRules = append(compiledRules, HighlightRule{re, rule.style})
    }

    e.languages["cobol"] = LanguageRules{
        name:  "COBOL",
        rules: compiledRules,
    }
}

func (e *Editor) currentBuffer() *Buffer {
    if len(e.buffers) == 0 {
        e.buffers = append(e.buffers, &Buffer{lines: []string{""}})
    }
    return e.buffers[e.activeBuf]
}

func (e *Editor) Init() error {
    s, err := tcell.NewScreen()
    if err != nil {
        return err
    }
    if err := s.Init(); err != nil {
        return err
    }
    e.screen = s
    return nil
}

func (e *Editor) Close() {
    if e.screen != nil {
        e.screen.Fini()
    }
}

func (e *Editor) OpenFile(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    // Detectar codificación
    var scanner *bufio.Scanner
    reader := bufio.NewReader(file)
    sample, err := reader.Peek(1024)
    if err != nil && err.Error() != "EOF" {
        return err
    }

    if !utf8.Valid(sample) {
        file.Seek(0, 0)
        decoder := charmap.ISO8859_1.NewDecoder()
        scanner = bufio.NewScanner(transform.NewReader(file, decoder))
    } else {
        file.Seek(0, 0)
        scanner = bufio.NewScanner(file)
    }

    // Leer contenido
    var lines []string
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }

    if len(lines) == 0 {
        lines = []string{""}
    }

    // Crear nuevo buffer
    buf := &Buffer{
        lines:    lines,
        filename: filename,
        saved:    true,
    }

    e.buffers = append(e.buffers, buf)
    e.activeBuf = len(e.buffers) - 1
    e.status = "Opened: " + filename

    return nil
}

func (e *Editor) SaveFile() error {
    buf := e.currentBuffer()
    if buf.filename == "" {
        e.status = "No filename set. Use 'Save As' feature (not implemented)"
        return nil
    }

    file, err := os.Create(buf.filename)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := bufio.NewWriter(file)
    for _, line := range buf.lines {
        _, err := writer.WriteString(line + "\n")
        if err != nil {
            return err
        }
    }
    writer.Flush()

    buf.saved = true
    e.status = "Saved: " + buf.filename
    return nil
}

func (e *Editor) NewTab() {
    e.buffers = append(e.buffers, &Buffer{
        lines: []string{""},
    })
    e.activeBuf = len(e.buffers) - 1
    e.status = "New tab created"
}

func (e *Editor) CloseTab() {
    if len(e.buffers) <= 1 {
        e.status = "Cannot close last tab"
        return
    }

    e.buffers = append(e.buffers[:e.activeBuf], e.buffers[e.activeBuf+1:]...)
    if e.activeBuf >= len(e.buffers) {
        e.activeBuf = len(e.buffers) - 1
    }
    e.status = "Tab closed"
}

func (e *Editor) SwitchTab(direction int) {
    e.activeBuf += direction
    if e.activeBuf < 0 {
        e.activeBuf = 0
    } else if e.activeBuf >= len(e.buffers) {
        e.activeBuf = len(e.buffers) - 1
    }
    buf := e.currentBuffer()
    if buf.filename != "" {
        e.status = buf.filename
    } else {
        e.status = "New buffer"
    }
}

func (e *Editor) DrawTabs() {
    width, _ := e.screen.Size()
    tabWidth := 20
    maxTabs := width / tabWidth

    for i, buf := range e.buffers {
        if i >= maxTabs {
            break
        }

        tabStyle := e.theme["tab_inactive"]
        if i == e.activeBuf {
            tabStyle = e.theme["tab_active"]
        }

        title := "Tab " + fmt.Sprint(i+1)
        if buf.filename != "" {
            name := buf.filename
            if strings.Contains(name, "/") {
                name = name[strings.LastIndex(name, "/")+1:]
            }
            if len(name) > tabWidth-4 {
                title = name[:tabWidth-4] + ".."
            } else {
                title = name
            }
        }

        // Dibujar pestaña
        for x := 0; x < tabWidth; x++ {
            posX := i*tabWidth + x
            if posX >= width {
                break
            }

            var ch rune
            if x < len(title) {
                ch = rune(title[x])
            } else {
                ch = ' '
            }

            e.screen.SetContent(posX, 0, ch, nil, tabStyle)
        }
    }
}

func (e *Editor) DrawLine(y int, line string) {
    width, _ := e.screen.Size()
    buf := e.currentBuffer()
    defaultStyle := e.theme["background"]

    // Determinar lenguaje basado en extensión
    var langRules LanguageRules
    var ok bool

    if e.highlight && buf.filename != "" {
        ext := strings.ToLower(buf.filename[strings.LastIndex(buf.filename, ".")+1:])
        switch ext {
        case "go":
            langRules, ok = e.languages["go"]
        case "c":
            langRules, ok = e.languages["c"]
        case "cob", "cbl", "cobol":
            langRules, ok = e.languages["cobol"]
        }
    }

    if !ok || !e.highlight {
        // Dibujar sin resaltado
        for x, ch := range line {
            if x >= width {
                break
            }
            e.screen.SetContent(x, y, ch, nil, defaultStyle)
        }
        return
    }

    // Convertir a runas para manejar Unicode correctamente
    runes := []rune(line)
    for x := 0; x < len(runes) && x < width; x++ {
        e.screen.SetContent(x, y, runes[x], nil, defaultStyle)
    }

    // Aplicar reglas de resaltado
    for _, rule := range langRules.rules {
        matches := rule.pattern.FindAllStringIndex(line, -1)
        for _, match := range matches {
            start := match[0]
            end := match[1]
            if start < 0 || end > len(runes) || start >= end {
                continue
            }

            for x := start; x < end && x < width; x++ {
                e.screen.SetContent(x, y, runes[x], nil, rule.style)
            }
        }
    }
}

func (e *Editor) Draw() {
    e.screen.Fill(' ', e.theme["background"])
    width, height := e.screen.Size()

    // Dibujar pestañas
    e.DrawTabs()

    // Dibujar contenido del buffer activo
    buf := e.currentBuffer()
    contentStartY := 1 // Espacio para pestañas

    for y := contentStartY; y < height-1; y++ {
        lineNum := y - contentStartY + buf.offsetY
        if lineNum < len(buf.lines) {
            line := buf.lines[lineNum]
            if buf.offsetX < len(line) {
                line = line[buf.offsetX:]
            } else {
                line = ""
            }
            e.DrawLine(y, line)
        }
    }

    // Dibujar cursor
    cursorScreenX := buf.cursorX - buf.offsetX
    cursorScreenY := contentStartY + buf.cursorY - buf.offsetY
    if cursorScreenX >= 0 && cursorScreenX < width &&
        cursorScreenY >= contentStartY && cursorScreenY < height-1 {
        e.screen.ShowCursor(cursorScreenX, cursorScreenY)
    }

    // Dibujar barra de estado
    status := e.status
    if len(status) > width {
        status = status[:width]
    }
    for x := 0; x < width; x++ {
        var ch rune
        if x < len(status) {
            ch = rune(status[x])
        } else {
            ch = ' '
        }
        e.screen.SetContent(x, height-1, ch, nil, e.theme["status"])
    }

    e.screen.Show()
}

func (e *Editor) handleEditKeys(ev *tcell.EventKey) {
    buf := e.currentBuffer()

    switch ev.Key() {
    case tcell.KeyUp:
        if buf.cursorY > 0 {
            buf.cursorY--
            if buf.cursorX > len(buf.lines[buf.cursorY]) {
                buf.cursorX = len(buf.lines[buf.cursorY])
            }
        }
    case tcell.KeyDown:
        if buf.cursorY < len(buf.lines)-1 {
            buf.cursorY++
            if buf.cursorX > len(buf.lines[buf.cursorY]) {
                buf.cursorX = len(buf.lines[buf.cursorY])
            }
        }
    case tcell.KeyLeft:
        if buf.cursorX > 0 {
            buf.cursorX--
        } else if buf.cursorY > 0 {
            buf.cursorY--
            buf.cursorX = len(buf.lines[buf.cursorY])
        }
    case tcell.KeyRight:
        if buf.cursorX < len(buf.lines[buf.cursorY]) {
            buf.cursorX++
        } else if buf.cursorY < len(buf.lines)-1 {
            buf.cursorY++
            buf.cursorX = 0
        }
    case tcell.KeyEnter:
        left := buf.lines[buf.cursorY][:buf.cursorX]
        right := buf.lines[buf.cursorY][buf.cursorX:]
        buf.lines[buf.cursorY] = left
        buf.lines = append(buf.lines[:buf.cursorY+1], append([]string{right}, buf.lines[buf.cursorY+1:]...)...)
        buf.cursorY++
        buf.cursorX = 0
        buf.saved = false
    case tcell.KeyBackspace, tcell.KeyBackspace2:
        if buf.cursorX > 0 {
            buf.lines[buf.cursorY] = buf.lines[buf.cursorY][:buf.cursorX-1] + buf.lines[buf.cursorY][buf.cursorX:]
            buf.cursorX--
            buf.saved = false
        } else if buf.cursorY > 0 {
            prevLineLen := len(buf.lines[buf.cursorY-1])
            buf.lines[buf.cursorY-1] += buf.lines[buf.cursorY]
            buf.lines = append(buf.lines[:buf.cursorY], buf.lines[buf.cursorY+1:]...)
            buf.cursorY--
            buf.cursorX = prevLineLen
            buf.saved = false
        }
    default:
        if ev.Key() == tcell.KeyRune {
            buf.lines[buf.cursorY] = buf.lines[buf.cursorY][:buf.cursorX] + string(ev.Rune()) + buf.lines[buf.cursorY][buf.cursorX:]
            buf.cursorX++
            buf.saved = false
        }
    }

    // Actualizar desplazamiento
    width, height := e.screen.Size()
    if buf.cursorX < buf.offsetX {
        buf.offsetX = buf.cursorX
    } else if buf.cursorX >= buf.offsetX+width {
        buf.offsetX = buf.cursorX - width + 1
    }

    if buf.cursorY < buf.offsetY {
        buf.offsetY = buf.cursorY
    } else if buf.cursorY >= buf.offsetY+(height-2) {
        buf.offsetY = buf.cursorY - (height - 2) + 1
    }
}

func (e *Editor) ProcessKey(ev *tcell.EventKey) {
    switch {
    case ev.Key() == tcell.KeyCtrlQ:
        e.quit = true
    case ev.Key() == tcell.KeyCtrlS:
        e.SaveFile()
    case ev.Key() == tcell.KeyCtrlT:
        e.NewTab()
    case ev.Key() == tcell.KeyCtrlW:
        e.CloseTab()
    case ev.Key() == tcell.KeyF1:
        e.highlight = !e.highlight
        if e.highlight {
            e.status = "Syntax highlighting: ON"
        } else {
            e.status = "Syntax highlighting: OFF"
        }
    case ev.Modifiers()&tcell.ModCtrl != 0 && ev.Key() == tcell.KeyLeft:
        e.SwitchTab(-1)
    case ev.Modifiers()&tcell.ModCtrl != 0 && ev.Key() == tcell.KeyRight:
        e.SwitchTab(1)
    default:
        e.handleEditKeys(ev)
    }
}

func (e *Editor) Run() {
    for !e.quit {
        e.Draw()
        ev := e.screen.PollEvent()
        switch ev := ev.(type) {
        case *tcell.EventKey:
            e.ProcessKey(ev)
        case *tcell.EventResize:
            e.screen.Sync()
        }
    }
}

func main() {
    editor := NewEditor()
    if err := editor.Init(); err != nil {
        fmt.Fprintf(os.Stderr, "Error initializing editor: %v\n", err)
        os.Exit(1)
    }
    defer editor.Close()

    if len(os.Args) > 1 {
        for _, filename := range os.Args[1:] {
            if err := editor.OpenFile(filename); err != nil {
                editor.status = "Error opening file: " + err.Error()
            }
        }
    }

    editor.Run()
}