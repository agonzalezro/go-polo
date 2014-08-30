package generator

import (
	"bufio"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/agonzalezro/polo/generator/utils"
	"github.com/russross/blackfriday"
)

type ParsedFile struct {
	Metadata map[string]string

	Author  string
	Title   string
	Slug    string
	Content string
	isPage  bool
	status  string // To keep track of the drafts
	summary string

	Category string
	tags     string
	Date     string

	scanner *bufio.Scanner
}

/*
 * Set the know metadata to the current object.
 *
 * The supported metadata for now is:
 *   - tags: going to be transformed to a []string
 *   - date: a string in ISO8601 format
 *   - slug: the slug for the url
 *
 * It's going to return false in case that the file doesn't have metadata.
 */
func (pf *ParsedFile) parseMetadata() bool {
	hasMetadata := false

	for pf.scanner.Scan() {
		line := pf.scanner.Text()

		metadataSplited := strings.Split(line, ":")
		key := strings.ToLower(metadataSplited[0])
		value := strings.Trim(strings.Join(metadataSplited[1:], ":"), " ")

		switch key {
		case "tags":
			// Remove all the spaces between comma and tag and
			// add one comma at the baginning other at the end, this will make the queryin much simpler
			pf.tags = fmt.Sprintf(",%s,", strings.Replace(value, " ", "", -1))
			hasMetadata = true
		case "date":
			pf.Date = value
			hasMetadata = true
		case "slug":
			pf.Slug = value
			hasMetadata = true
		case "title":
			pf.Title = value
			hasMetadata = true
		case "status":
			pf.status = value
			hasMetadata = true
		case "summary":
			pf.summary = value
			hasMetadata = true
		case "author":
			pf.Author = value
			hasMetadata = true
		default:
			return hasMetadata
		}
	}

	return hasMetadata
}

// Loads the content of the file object from the given filename.
func (pf *ParsedFile) load(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	pf.scanner = bufio.NewScanner(file)
	if hasMetadata := pf.parseMetadata(); !hasMetadata {
		// Rewind the file and reset the scanner
		file.Seek(0, 0)
		pf.scanner = bufio.NewScanner(file)
	}

	isFirstLine := true
	for pf.scanner.Scan() {
		line := pf.scanner.Text()
		if isFirstLine == true {
			if line == "" {
				// Do not read empty lines at the beginning
				continue
			}

			if pf.Title == "" {
				pf.Title = line
			}
			if pf.Slug == "" {
				pf.Slug = utils.Slugify(line)
			}
			pf.scanner.Scan() // We don't want the title underlining

			isFirstLine = false
		} else {
			pf.Content += line + "\n"
		}
	}

	// Set the category from the filePath
	splittedPath := strings.Split(filePath, "/")
	length := len(splittedPath)
	if length > 1 {
		pf.Category = splittedPath[len(splittedPath)-2]
	}

	return nil
}

// Split the tags into a list.
func (pf ParsedFile) Tags() []string {
	return strings.Split(pf.tags, ",")
}

// Function to be called from the templates. It render safe HTML code.
func (file ParsedFile) Html(content string) template.HTML {
	// set up the HTML renderer
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	// set up the parser
	extensions := 0
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS

	html := blackfriday.Markdown([]byte(content), renderer, extensions)
	return template.HTML(html)
}

func (file ParsedFile) Summary() string {
	if file.summary != "" {
		return file.summary
	}
	// Avoid empty lines
	for _, content := range strings.Split(file.Content, "\n\n") {
		if content != "" {
			return content
		}
	}
	return ""
}

// Store the file in a "permanent" storage.
func (file ParsedFile) save(db *DB) error {
	query := `
    INSERT INTO files (author, title, slug, content, category, tags, date, status, summary, is_page)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	// SQLite doesn't support booleans :(
	isPageInt := 0
	if file.isPage {
		isPageInt = 1
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	if _, err := tx.Exec(query, file.Author, file.Title, file.Slug, file.Content, file.Category, file.tags, file.Date, file.status, file.summary, isPageInt); err != nil {
		return err
	}
	return nil
}
