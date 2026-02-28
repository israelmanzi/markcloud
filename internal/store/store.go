package store

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Document struct {
	Path        string
	Title       string
	ContentMD   string
	ContentHTML string
	SHA         string
	Public      bool
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SearchResult struct {
	Path    string
	Title   string
	Snippet string
}

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			path TEXT PRIMARY KEY,
			title TEXT NOT NULL DEFAULT '',
			content_md TEXT NOT NULL DEFAULT '',
			content_html TEXT NOT NULL DEFAULT '',
			sha TEXT NOT NULL DEFAULT '',
			public BOOLEAN NOT NULL DEFAULT 0,
			tags TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
			path, title, content_md,
			content='documents',
			content_rowid='rowid'
		);

		CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
			INSERT INTO documents_fts(rowid, path, title, content_md)
			VALUES (new.rowid, new.path, new.title, new.content_md);
		END;

		CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, path, title, content_md)
			VALUES ('delete', old.rowid, old.path, old.title, old.content_md);
		END;

		CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, path, title, content_md)
			VALUES ('delete', old.rowid, old.path, old.title, old.content_md);
			INSERT INTO documents_fts(rowid, path, title, content_md)
			VALUES (new.rowid, new.path, new.title, new.content_md);
		END;
	`)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Upsert(doc *Document) error {
	tags, _ := json.Marshal(doc.Tags)
	now := time.Now()

	_, err := s.db.Exec(`
		INSERT INTO documents (path, title, content_md, content_html, sha, public, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			title=excluded.title,
			content_md=excluded.content_md,
			content_html=excluded.content_html,
			sha=excluded.sha,
			public=excluded.public,
			tags=excluded.tags,
			updated_at=excluded.updated_at
	`, doc.Path, doc.Title, doc.ContentMD, doc.ContentHTML, doc.SHA, doc.Public, string(tags), now, now)
	return err
}

func (s *Store) Get(path string) (*Document, error) {
	doc := &Document{}
	var tags string
	err := s.db.QueryRow(`
		SELECT path, title, content_md, content_html, sha, public, tags, created_at, updated_at
		FROM documents WHERE path = ?
	`, path).Scan(&doc.Path, &doc.Title, &doc.ContentMD, &doc.ContentHTML, &doc.SHA, &doc.Public, &tags, &doc.CreatedAt, &doc.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(tags), &doc.Tags)
	return doc, nil
}

func (s *Store) List(prefix string) ([]Document, error) {
	rows, err := s.db.Query(`
		SELECT path, title, sha, public, tags, updated_at
		FROM documents WHERE path LIKE ? ORDER BY path
	`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		var tags string
		if err := rows.Scan(&d.Path, &d.Title, &d.SHA, &d.Public, &tags, &d.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tags), &d.Tags)
		docs = append(docs, d)
	}
	return docs, nil
}

func (s *Store) Delete(path string) error {
	_, err := s.db.Exec("DELETE FROM documents WHERE path = ?", path)
	return err
}

func (s *Store) Search(query string) ([]SearchResult, error) {
	rows, err := s.db.Query(`
		SELECT path, title, snippet(documents_fts, 2, '<mark>', '</mark>', '...', 30)
		FROM documents_fts WHERE documents_fts MATCH ?
		ORDER BY rank
	`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Path, &r.Title, &r.Snippet); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func (s *Store) GetAllSHAs() (map[string]string, error) {
	rows, err := s.db.Query("SELECT path, sha FROM documents")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shas := make(map[string]string)
	for rows.Next() {
		var path, sha string
		if err := rows.Scan(&path, &sha); err != nil {
			return nil, err
		}
		shas[path] = sha
	}
	return shas, nil
}

func (s *Store) DeleteAllExcept(paths []string) error {
	if len(paths) == 0 {
		_, err := s.db.Exec("DELETE FROM documents")
		return err
	}
	args := make([]any, len(paths))
	placeholders := ""
	for i, p := range paths {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = p
	}
	_, err := s.db.Exec("DELETE FROM documents WHERE path NOT IN ("+placeholders+")", args...)
	return err
}
