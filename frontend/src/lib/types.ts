// Shapes of the backend API, see docs/API.md.

export type PostStatus = 'draft' | 'unlisted' | 'published';

export const POST_STATUSES: PostStatus[] = ['draft', 'unlisted', 'published'];

export interface List<T> {
	items: T[];
	total: number;
}

export interface PostSummary {
	slug: string;
	title: string;
	published_at: string;
}

export interface Post {
	slug: string;
	title: string;
	body: string;
	status: PostStatus;
	published_at: string | null;
	updated_at: string;
}

export interface Comment {
	id: number;
	author: string | null;
	body: string;
	created_at: string;
}

export interface Page {
	slug: string;
	title: string;
	body: string;
	updated_at: string;
}

export interface NavLink {
	label: string;
	url: string;
}

export interface Nav {
	header: NavLink[];
	footer: NavLink[];
}

export interface FileMeta {
	id: string;
	name: string;
	content_type: string;
	size: number;
	url: string;
}

export interface AdminUser {
	id: string;
	login: string;
}

export interface AdminSession {
	id: string;
	user_agent: string;
	ip: string;
	created_at: string;
	last_used_at: string;
	current: boolean;
}

export interface AdminPostSummary {
	id: string;
	slug: string;
	title: string;
	status: PostStatus;
	created_at: string;
	updated_at: string;
	published_at: string | null;
}

export interface AdminPost extends AdminPostSummary {
	body: string;
}

export interface AdminPageSummary {
	id: string;
	slug: string;
	title: string;
	created_at: string;
	updated_at: string;
}

export interface AdminPage extends AdminPageSummary {
	body: string;
}

export interface AdminNavItem {
	id: number;
	label: string;
	url: string;
}

export interface AdminNav {
	header: AdminNavItem[];
	footer: AdminNavItem[];
}

export interface AdminFile {
	id: string;
	name: string;
	content_type: string;
	size: number;
	created_at: string;
	url: string;
}

export type CommentStatus = 'pending' | 'approved' | 'rejected';

export interface AdminComment {
	id: number;
	post_slug: string;
	post_title: string;
	author: string | null;
	body: string;
	status: CommentStatus;
	created_at: string;
}

export interface StatsItem {
	path: string;
	count: number;
}
