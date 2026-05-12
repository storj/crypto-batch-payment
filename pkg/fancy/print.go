package fancy

import (
	"fmt"
	"io"

	"github.com/logrusorgru/aurora"
)

var (
	Info  = aurora.White
	Warn  = aurora.Yellow
	Error = aurora.Red
)

type Level = func(arg any) aurora.Value

func Println(level Level, args ...any) {
	fmt.Println(level(fmt.Sprint(args...)))
}

func Printf(level Level, format string, args ...any) {
	fmt.Print(level(fmt.Sprintf(format, args...)))
}

func Infoln(args ...any) {
	Println(Info, args...)
}

func Infof(format string, args ...any) {
	Printf(Info, format, args...)
}

func Warnln(args ...any) {
	Println(Warn, args...)
}

func Warnf(format string, args ...any) {
	Printf(Warn, format, args...)
}

func Errorln(args ...any) {
	Println(Error, args...)
}

func Errorf(format string, args ...any) {
	Printf(Error, format, args...)
}

func Fprintln(w io.Writer, level Level, args ...any) {
	_, _ = fmt.Fprintln(w, level(fmt.Sprint(args...)))
}

func Fprintf(w io.Writer, level Level, format string, args ...any) {
	_, _ = fmt.Fprint(w, level(fmt.Sprintf(format, args...)))
}

func Finfoln(w io.Writer, args ...any) {
	Fprintln(w, Info, args...)
}

func Finfof(w io.Writer, format string, args ...any) {
	Fprintf(w, Info, format, args...)
}

func Fwarnln(w io.Writer, args ...any) {
	Fprintln(w, Warn, args...)
}

func Fwarnf(w io.Writer, format string, args ...any) {
	Fprintf(w, Warn, format, args...)
}

func Ferrorln(w io.Writer, args ...any) {
	Fprintln(w, Error, args...)
}

func Ferrorf(w io.Writer, format string, args ...any) {
	Fprintf(w, Error, format, args...)
}
