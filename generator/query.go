package generator

import (
	"fmt"
)

type QueryError error

// getQueryInterface is going to cast the array of string to an array of
// interfaces adding the default parameter isPage in the position 0
func getQueryInterface(isPage bool, args ...string) []interface{} {
	// This funny casting is needed to call the sql .Query()
	sqlArgs := make([]interface{}, len(args)+1)
	sqlArgs[0] = 0
	if isPage {
		sqlArgs[0] = 1
	}
	for i, v := range args {
		sqlArgs[i+1] = interface{}(v)
	}
	return sqlArgs
}

func (site Site) Query(isPage bool, where string, page int, args ...string) (files []*ParsedFile, err error) {
	// In case that a where clausule needs to be added, add the AND at the beginning
	if where != "" {
		where = fmt.Sprintf("AND %s", where)
	}
	limitAndOffset := ""
	if page >= 0 {
		limitAndOffset = fmt.Sprintf("LIMIT %d OFFSET %d", site.Config.PaginationSize, (page-1)*site.Config.PaginationSize)
	}
	query := fmt.Sprintf(`
        SELECT author, title, slug, content, category, tags, date, summary
        FROM files
        WHERE is_page = ?
        AND status != 'draft'
        %s
        ORDER BY datetime(date) DESC
        %s
    `, where, limitAndOffset)

	sqlArgs := getQueryInterface(isPage, args...)

	cache_key := fmt.Sprintf("%s%v", query, sqlArgs)
	if files, ok := site.Cache[cache_key]; ok {
		return files.([]*ParsedFile), nil
	}

	rows, err := site.db.Query(query, sqlArgs...)
	if err != nil {
		return nil, QueryError(err)
	}

	for rows.Next() {
		file := &ParsedFile{isPage: isPage}
		rows.Scan(&file.Author, &file.Title, &file.Slug, &file.Content, &file.Category, &file.tags, &file.Date, &file.summary)
		files = append(files, file)
	}

	site.Cache[cache_key] = files
	return files, nil
}

func (site Site) QueryArticles(where string, page int, args ...string) ([]*ParsedFile, error) {
	return site.Query(false, where, page, args...)
}

func (site Site) QueryPages() ([]*ParsedFile, error) {
	return site.Query(true, "", -1)
}
