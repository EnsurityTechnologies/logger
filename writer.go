package logger

import (
	"bytes"
	"io"
)

type writer struct {
	b     bytes.Buffer
	w     []io.Writer
	color []ColorOption
}

func newWriter(w []io.Writer, color []ColorOption) *writer {
	return &writer{w: w, color: color}
}

func (w *writer) UpdateWriter(nw io.Writer) {
	tw := make([]io.Writer, 0)
	color := make([]ColorOption, 0)
	for i, wr := range w.w {
		_, ok := wr.(*fileWrite)
		if !ok {
			tw = append(tw, wr)
			color = append(color, w.color[i])
		}
	}
	tw = append(tw, nw)
	color = append(color, ColorOff)
	w.w = tw
	w.color = color
}

func (w *writer) Flush(level Level) (err error) {
	var unwritten = w.b.Bytes()

	for i, wr := range w.w {
		if lw, ok := wr.(LevelWriter); ok {
			_, err = lw.LevelWrite(level, unwritten)
		} else {
			l := len(unwritten)
			if unwritten[0] == 27 {
				unwritten = unwritten[5 : l-4]
			}
			if w.color[i] != ColorOff {
				color := _levelToColor[level]
				colorbytes := []byte(color.Sprintf("%s", unwritten))
				_, err = wr.Write(colorbytes)
			} else {
				_, err = wr.Write(unwritten)
			}

		}
	}

	w.b.Reset()
	return err
}

func (w *writer) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

func (w *writer) WriteByte(c byte) error {
	return w.b.WriteByte(c)
}

func (w *writer) WriteString(s string) (int, error) {
	return w.b.WriteString(s)
}

type LevelWriter interface {
	LevelWrite(level Level, p []byte) (n int, err error)
}

type LeveledWriter struct {
	standard  io.Writer
	overrides map[Level]io.Writer
}

func NewLeveledWriter(standard io.Writer, overrides map[Level]io.Writer) *LeveledWriter {
	return &LeveledWriter{
		standard:  standard,
		overrides: overrides,
	}
}

func (lw *LeveledWriter) Write(p []byte) (int, error) {
	return lw.standard.Write(p)
}

func (lw *LeveledWriter) LevelWrite(level Level, p []byte) (int, error) {
	w, ok := lw.overrides[level]
	if !ok {
		w = lw.standard
	}
	return w.Write(p)
}
