<script lang="ts">
	import { AUTHOR_MAX, BODY_MAX, commentPayload, commentResult } from '$lib/comments';

	interface Props {
		slug: string;
		/** Result of the no-JS form action fallback, if any. */
		fallback?: { ok: boolean; message: string } | null;
	}

	let { slug, fallback = null }: Props = $props();

	let author = $state('');
	let body = $state('');
	let website = $state('');
	let sending = $state(false);
	let result = $state<{ ok: boolean; message: string } | null>(null);
	let shown = $derived(result ?? fallback);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (sending) return;
		sending = true;
		result = null;
		try {
			const res = await fetch(`/api/posts/${encodeURIComponent(slug)}/comments`, {
				method: 'POST',
				headers: { 'content-type': 'application/json', accept: 'application/json' },
				body: JSON.stringify(commentPayload(author, body, website))
			});
			let error: string | undefined;
			if (!res.ok) {
				try {
					error = (await res.json()).error;
				} catch {
					// no JSON body
				}
			}
			result = commentResult(res.status, error);
			if (result.ok) {
				author = '';
				body = '';
			}
		} catch {
			result = { ok: false, message: 'Network error. Please check your connection and try again.' };
		} finally {
			sending = false;
		}
	}
</script>

<form method="POST" action="?/comment" onsubmit={submit}>
	<div class="row">
		<label>
			Name (optional)
			<input
				name="author"
				type="text"
				maxlength={AUTHOR_MAX}
				autocomplete="nickname"
				bind:value={author}
			/>
		</label>
	</div>
	<label>
		Comment
		<textarea name="body" rows="5" maxlength={BODY_MAX} required bind:value={body}></textarea>
	</label>
	<div class="hp" aria-hidden="true">
		<label>
			Website
			<input name="website" type="text" tabindex="-1" autocomplete="off" bind:value={website} />
		</label>
	</div>
	{#if shown}
		<p class="message {shown.ok ? 'ok' : 'error'}" role="status">{shown.message}</p>
	{/if}
	<div class="actions">
		<button type="submit" aria-busy={sending} disabled={sending}>Send comment</button>
		<small>Comments are moderated. One comment per day.</small>
	</div>
</form>

<style>
	.hp {
		position: absolute;
		left: -10000px;
		width: 1px;
		height: 1px;
		overflow: hidden;
	}

	.row {
		max-width: 20rem;
	}

	textarea {
		resize: vertical;
	}

	.message {
		padding: 0.6rem 0.9rem;
		border-radius: var(--radius-sm);
		font-size: 0.9375rem;
	}

	.ok {
		background: var(--success-soft);
		color: var(--success);
	}

	.error {
		background: var(--danger-soft);
		color: var(--danger);
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.75rem 1rem;
	}

	.actions button {
		width: auto;
		margin: 0;
	}

	small {
		color: var(--muted);
	}
</style>
