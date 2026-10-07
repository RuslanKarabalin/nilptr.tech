// Messages for POST /api/posts/{slug}/comments results, shared by the
// browser fetch and the no-JS form action fallback.

export const AUTHOR_MAX = 64;
export const BODY_MAX = 4000;

export interface CommentResult {
	ok: boolean;
	message: string;
}

export function commentResult(status: number, error?: string): CommentResult {
	if (status === 201) {
		return { ok: true, message: 'Thank you! Your comment will appear after moderation.' };
	}
	if (status === 429) {
		return {
			ok: false,
			message: 'You have already left a comment today. Please try again tomorrow.'
		};
	}
	if (status === 404) {
		return { ok: false, message: 'Comments are not available for this post.' };
	}
	if (status >= 400 && status < 500) {
		return { ok: false, message: error || 'Please check your comment and try again.' };
	}
	return {
		ok: false,
		message: `Could not send the comment (status ${status}). Please try again later.`
	};
}

export function commentPayload(author: string, body: string, website: string) {
	const trimmed = author.trim();
	return {
		...(trimmed ? { author: trimmed } : {}),
		body: body.trim(),
		website
	};
}
