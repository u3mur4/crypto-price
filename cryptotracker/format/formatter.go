package format

// Formatter displays the markets to the user
type Formatter interface {
	Open()
	Show(m Market)
	Close()
}
