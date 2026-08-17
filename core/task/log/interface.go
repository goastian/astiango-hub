package log

type Driver interface {
	Init()
	Close() (err error)
	WriteLine(id string, line string) (err error)
	WriteLines(id string, lines []string) (err error)
	Find(id string, pattern string, skip int, limit int, tail bool) (lines []string, err error)
	Count(id string, pattern string) (n int, err error)
}
