import { fail } from '@sveltejs/kit';
import {
	actionFailed,
	adminApi,
	field,
	loadFailed,
	pageCount,
	pagination
} from '$lib/server/admin';
import { errorMessage } from '$lib/server/backend';
import type { AdminFile, List } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const { page, limit, offset } = pagination(event.url, 50);
	try {
		const files = await adminApi(event).get<List<AdminFile>>('/files', { limit, offset });
		return { files: files.items, total: files.total, page, pages: pageCount(files.total, limit) };
	} catch (err) {
		loadFailed(err);
	}
};

export const actions: Actions = {
	// Streams the multipart body to the backend without buffering it here.
	// The form must contain only the `file` field. adapter-node limits
	// request bodies with BODY_SIZE_LIMIT, set it to the upload limit.
	upload: async (event) => {
		const contentType = event.request.headers.get('content-type') ?? '';
		if (!contentType.toLowerCase().startsWith('multipart/form-data') || !event.request.body) {
			return fail(400, { error: 'Choose a file to upload.' });
		}
		try {
			const res = await adminApi(event).raw('/files', {
				method: 'POST',
				headers: { 'content-type': contentType, accept: 'application/json' },
				body: event.request.body,
				timeout: 0,
				// required by Node fetch for streaming request bodies
				duplex: 'half'
			} as RequestInit);
			if (res.status === 401) {
				await res.body?.cancel();
				return fail(401, { error: 'Session expired, please log in again.' });
			}
			if (!res.ok) return fail(res.status, { error: await errorMessage(res) });
			const uploaded = (await res.json()) as AdminFile;
			return { success: `Uploaded ${uploaded.name}.`, uploaded };
		} catch (err) {
			return actionFailed(err);
		}
	},
	delete: async (event) => {
		const id = field(await event.request.formData(), 'id');
		try {
			await adminApi(event).del(`/files/${encodeURIComponent(id)}`);
			return { success: 'File deleted.' };
		} catch (err) {
			return actionFailed(err);
		}
	}
};
