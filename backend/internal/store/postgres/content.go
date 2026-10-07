package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

// ---- posts ----

const postColumns = `id, slug, title, body, status::text, created_at, updated_at, published_at`

func scanPost(row pgx.Row) (store.Post, error) {
	var p store.Post
	err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Body, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.PublishedAt)
	p.CreatedAt, p.UpdatedAt, p.PublishedAt = utc(p.CreatedAt), utc(p.UpdatedAt), utcPtr(p.PublishedAt)
	return p, err
}

func collectPosts(rows pgx.Rows) ([]store.Post, error) {
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (store.Post, error) { return scanPost(r) })
	return out, mapErr(err)
}

// PublishedPosts lists published posts, newest first. Bodies are omitted.
func (db *DB) PublishedPosts(ctx context.Context, pg store.Page) ([]store.Post, int, error) {
	var total int
	if err := db.pool.QueryRow(ctx, `select count(*) from posts where status = 'published'`).Scan(&total); err != nil {
		return nil, 0, mapErr(err)
	}
	rows, err := db.pool.Query(ctx, `
		select id, slug, title, '', status::text, created_at, updated_at, published_at
		from posts where status = 'published'
		order by published_at desc, id
		limit $1 offset $2`, pg.Limit, pg.Offset)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	items, err := collectPosts(rows)
	return items, total, err
}

// VisiblePostBySlug returns a published or unlisted post.
func (db *DB) VisiblePostBySlug(ctx context.Context, slug string) (store.Post, error) {
	p, err := scanPost(db.pool.QueryRow(ctx,
		`select `+postColumns+` from posts where slug = $1 and status in ('published', 'unlisted')`, slug))
	return p, mapErr(err)
}

// AdminPosts lists all posts, newest first by updated_at. Bodies are omitted.
func (db *DB) AdminPosts(ctx context.Context, status string, pg store.Page) ([]store.Post, int, error) {
	var total int
	err := db.pool.QueryRow(ctx,
		`select count(*) from posts where $1 = '' or status::text = $1`, status).Scan(&total)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	rows, err := db.pool.Query(ctx, `
		select id, slug, title, '', status::text, created_at, updated_at, published_at
		from posts where $1 = '' or status::text = $1
		order by updated_at desc, id
		limit $2 offset $3`, status, pg.Limit, pg.Offset)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	items, err := collectPosts(rows)
	return items, total, err
}

func (db *DB) PostByID(ctx context.Context, id uuid.UUID) (store.Post, error) {
	p, err := scanPost(db.pool.QueryRow(ctx, `select `+postColumns+` from posts where id = $1`, id))
	return p, mapErr(err)
}

func (db *DB) CreatePost(ctx context.Context, in store.PostInput) (store.Post, error) {
	p, err := scanPost(db.pool.QueryRow(ctx, `
		insert into posts (slug, title, body, status, published_at)
		values ($1, $2, $3, $4::post_status, case when $4 = 'published' then now() end)
		returning `+postColumns, in.Slug, in.Title, in.Body, in.Status))
	return p, mapErr(err)
}

func (db *DB) UpdatePost(ctx context.Context, id uuid.UUID, in store.PostInput) (store.Post, error) {
	p, err := scanPost(db.pool.QueryRow(ctx, `
		update posts set
			slug = $2, title = $3, body = $4, status = $5::post_status, updated_at = now(),
			published_at = coalesce(published_at, case when $5 = 'published' then now() end)
		where id = $1
		returning `+postColumns, id, in.Slug, in.Title, in.Body, in.Status))
	return p, mapErr(err)
}

func (db *DB) DeletePost(ctx context.Context, id uuid.UUID) error {
	return deleteByID(ctx, db, `delete from posts where id = $1`, id)
}

func deleteByID(ctx context.Context, db *DB, sql string, id any) error {
	tag, err := db.pool.Exec(ctx, sql, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// ---- pages ----

const pageColumns = `id, slug, title, body, updated_at`

func scanPage(row pgx.Row) (store.PageDoc, error) {
	var p store.PageDoc
	err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Body, &p.UpdatedAt)
	p.UpdatedAt = utc(p.UpdatedAt)
	return p, mapErr(err)
}

func (db *DB) PageBySlug(ctx context.Context, slug string) (store.PageDoc, error) {
	return scanPage(db.pool.QueryRow(ctx, `select `+pageColumns+` from pages where slug = $1`, slug))
}

func (db *DB) PageByID(ctx context.Context, id uuid.UUID) (store.PageDoc, error) {
	return scanPage(db.pool.QueryRow(ctx, `select `+pageColumns+` from pages where id = $1`, id))
}

func (db *DB) AdminPages(ctx context.Context, pg store.Page) ([]store.PageDoc, int, error) {
	var total int
	if err := db.pool.QueryRow(ctx, `select count(*) from pages`).Scan(&total); err != nil {
		return nil, 0, mapErr(err)
	}
	rows, err := db.pool.Query(ctx, `
		select id, slug, title, '', updated_at from pages
		order by updated_at desc, id limit $1 offset $2`, pg.Limit, pg.Offset)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (store.PageDoc, error) { return scanPage(r) })
	return items, total, mapErr(err)
}

func (db *DB) CreatePage(ctx context.Context, in store.PageInput) (store.PageDoc, error) {
	return scanPage(db.pool.QueryRow(ctx, `
		insert into pages (slug, title, body) values ($1, $2, $3)
		returning `+pageColumns, in.Slug, in.Title, in.Body))
}

func (db *DB) UpdatePage(ctx context.Context, id uuid.UUID, in store.PageInput) (store.PageDoc, error) {
	return scanPage(db.pool.QueryRow(ctx, `
		update pages set slug = $2, title = $3, body = $4, updated_at = now()
		where id = $1 returning `+pageColumns, id, in.Slug, in.Title, in.Body))
}

func (db *DB) DeletePage(ctx context.Context, id uuid.UUID) error {
	return deleteByID(ctx, db, `delete from pages where id = $1`, id)
}

// ---- navigation ----

func (db *DB) NavItems(ctx context.Context) ([]store.NavItem, error) {
	rows, err := db.pool.Query(ctx,
		`select id, place::text, label, url from nav_items order by place, position`)
	if err != nil {
		return nil, mapErr(err)
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (store.NavItem, error) {
		var n store.NavItem
		err := r.Scan(&n.ID, &n.Place, &n.Label, &n.URL)
		return n, err
	})
	return items, mapErr(err)
}

// ReplaceNav replaces all items; the position is the index within a place.
func (db *DB) ReplaceNav(ctx context.Context, items []store.NavItem) error {
	return pgx.BeginFunc(ctx, db.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `delete from nav_items`); err != nil {
			return mapErr(err)
		}
		pos := map[string]int{}
		batch := &pgx.Batch{}
		for _, it := range items {
			batch.Queue(`insert into nav_items (place, label, url, position) values ($1::nav_place, $2, $3, $4)`,
				it.Place, it.Label, it.URL, pos[it.Place])
			pos[it.Place]++
		}
		return mapErr(tx.SendBatch(ctx, batch).Close())
	})
}

// ---- files ----

const fileColumns = `id, name, content_type, size, s3_key, created_at`

func scanFile(row pgx.Row) (store.File, error) {
	var f store.File
	err := row.Scan(&f.ID, &f.Name, &f.ContentType, &f.Size, &f.S3Key, &f.CreatedAt)
	f.CreatedAt = utc(f.CreatedAt)
	return f, mapErr(err)
}

func (db *DB) FileByID(ctx context.Context, id uuid.UUID) (store.File, error) {
	return scanFile(db.pool.QueryRow(ctx, `select `+fileColumns+` from files where id = $1`, id))
}

func (db *DB) Files(ctx context.Context, pg store.Page) ([]store.File, int, error) {
	var total int
	if err := db.pool.QueryRow(ctx, `select count(*) from files`).Scan(&total); err != nil {
		return nil, 0, mapErr(err)
	}
	rows, err := db.pool.Query(ctx, `select `+fileColumns+` from files
		order by created_at desc, id limit $1 offset $2`, pg.Limit, pg.Offset)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (store.File, error) { return scanFile(r) })
	return items, total, mapErr(err)
}

func (db *DB) CreateFile(ctx context.Context, f store.File) (store.File, error) {
	return scanFile(db.pool.QueryRow(ctx, `
		insert into files (id, name, content_type, size, s3_key) values ($1, $2, $3, $4, $5)
		returning `+fileColumns, f.ID, f.Name, f.ContentType, f.Size, f.S3Key))
}

func (db *DB) DeleteFile(ctx context.Context, id uuid.UUID) error {
	return deleteByID(ctx, db, `delete from files where id = $1`, id)
}

// ---- comments ----

func (db *DB) ApprovedComments(ctx context.Context, postID uuid.UUID) ([]store.Comment, error) {
	rows, err := db.pool.Query(ctx, `
		select id, author, body, created_at from comments
		where post_id = $1 and status = 'approved'
		order by created_at, id`, postID)
	if err != nil {
		return nil, mapErr(err)
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (store.Comment, error) {
		var c store.Comment
		err := r.Scan(&c.ID, &c.Author, &c.Body, &c.CreatedAt)
		c.CreatedAt = utc(c.CreatedAt)
		return c, err
	})
	return items, mapErr(err)
}

// CreateComment inserts a pending comment. It returns store.ErrConflict
// when a comment with the same ip_hash exists (daily limit).
func (db *DB) CreateComment(ctx context.Context, c store.NewComment) error {
	_, err := db.pool.Exec(ctx,
		`insert into comments (post_id, author, body, ip_hash) values ($1, $2, $3, $4)`,
		c.PostID, c.Author, c.Body, c.IPHash)
	return mapErr(err)
}

func (db *DB) AdminComments(ctx context.Context, status string, pg store.Page) ([]store.Comment, int, error) {
	var total int
	err := db.pool.QueryRow(ctx,
		`select count(*) from comments where $1 = '' or status::text = $1`, status).Scan(&total)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	rows, err := db.pool.Query(ctx, `
		select c.id, p.slug, p.title, c.author, c.body, c.status::text, c.created_at
		from comments c join posts p on p.id = c.post_id
		where $1 = '' or c.status::text = $1
		order by c.created_at desc, c.id desc
		limit $2 offset $3`, status, pg.Limit, pg.Offset)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (store.Comment, error) {
		var c store.Comment
		err := r.Scan(&c.ID, &c.PostSlug, &c.PostTitle, &c.Author, &c.Body, &c.Status, &c.CreatedAt)
		c.CreatedAt = utc(c.CreatedAt)
		return c, err
	})
	return items, total, mapErr(err)
}

func (db *DB) SetCommentStatus(ctx context.Context, id int64, status string) error {
	tag, err := db.pool.Exec(ctx, `update comments set status = $2::comment_status where id = $1`, id, status)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (db *DB) DeleteComment(ctx context.Context, id int64) error {
	return deleteByID(ctx, db, `delete from comments where id = $1`, id)
}

// ---- views ----

func (db *DB) AddView(ctx context.Context, path string) error {
	_, err := db.pool.Exec(ctx, `insert into page_views (path) values ($1)`, path)
	return mapErr(err)
}

// ViewStats counts views per path in [from, to).
func (db *DB) ViewStats(ctx context.Context, from, to time.Time) ([]store.PathCount, error) {
	rows, err := db.pool.Query(ctx, `
		select path, count(*) from page_views
		where created_at >= $1 and created_at < $2
		group by path order by count(*) desc, path`, from, to)
	if err != nil {
		return nil, mapErr(err)
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (store.PathCount, error) {
		var pc store.PathCount
		err := r.Scan(&pc.Path, &pc.Count)
		return pc, err
	})
	return items, mapErr(err)
}
