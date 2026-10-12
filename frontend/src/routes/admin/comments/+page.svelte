<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Icon from '$lib/components/Icon.svelte';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';
	import Pager from '$lib/components/admin/Pager.svelte';
	import { formatDateTime } from '$lib/format';

	let { data, form } = $props();
</script>

<svelte:head>
	<title>Comments - nilptr admin</title>
</svelte:head>

<h1>Comments</h1>

<nav class="tabs" aria-label="Comment status">
	{#each data.statuses as status (status)}
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
		<a href="?status={status}" aria-current={status === data.status ? 'page' : undefined}>
			{status}
		</a>
	{/each}
</nav>

<FormMessage error={form?.error} success={form && 'success' in form ? form.success : null} />

{#if data.comments.length === 0}
	<div class="empty">
		<Icon name="message" size="2rem" />
		<p>No {data.status} comments.</p>
	</div>
{:else}
	<ul class="list">
		{#each data.comments as comment (comment.id)}
			<li class="comment">
				<span class="avatar" aria-hidden="true"
					>{(comment.author?.trim()[0] ?? '?').toUpperCase()}</span
				>
				<div class="content">
					<div class="meta">
						<strong>{comment.author || 'Anonymous'}</strong>
						<span class="badge badge-{comment.status}">{comment.status}</span>
						<span class="when">{formatDateTime(comment.created_at)}</span>
					</div>
					<p class="on">
						on <a href={resolve('/posts/[slug]', { slug: comment.post_slug })}
							>{comment.post_title}</a
						>
					</p>
					<p class="body">{comment.body}</p>
					<form method="POST" use:enhance class="actions">
						<input type="hidden" name="id" value={comment.id} />
						{#if comment.status !== 'approved'}
							<button type="submit" formaction="?/approve">Approve</button>
						{/if}
						{#if comment.status !== 'rejected'}
							<button type="submit" formaction="?/reject" class="secondary">Reject</button>
						{/if}
						<button type="submit" formaction="?/delete" class="secondary outline delete">
							Delete
						</button>
					</form>
				</div>
			</li>
		{/each}
	</ul>
	<p class="total">{data.total} {data.status} comments</p>
{/if}

<Pager page={data.page} pages={data.pages} />

<style>
	.tabs {
		display: inline-flex;
		gap: 0.25rem;
		margin-bottom: 1.5rem;
		padding: 0.25rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
	}

	.tabs a {
		padding: 0.35rem 0.9rem;
		border-radius: 999px;
		color: var(--muted);
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
		text-transform: capitalize;
	}

	.tabs a:hover {
		color: var(--heading);
	}

	.tabs a[aria-current='page'] {
		background: var(--accent);
		color: var(--accent-contrast);
	}

	.list {
		display: grid;
		gap: 0.75rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.comment {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		gap: 1rem;
		margin: 0;
		padding: 1.25rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		box-shadow: var(--shadow-sm);
		list-style: none;
	}

	.avatar {
		display: grid;
		place-items: center;
		width: 2.4rem;
		height: 2.4rem;
		border-radius: 50%;
		background: var(--accent-soft);
		color: var(--accent);
		font-weight: 700;
	}

	.meta {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.4rem 0.75rem;
	}

	.meta strong {
		color: var(--heading);
	}

	.when,
	.on,
	.total {
		color: var(--muted);
		font-size: 0.875rem;
	}

	.on {
		margin: 0.15rem 0 0.6rem;
	}

	.body {
		margin-bottom: 1rem;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin: 0;
	}

	.actions button {
		width: auto;
		margin: 0;
		padding: 0.4rem 0.9rem;
		font-size: 0.875rem;
	}

	.delete:hover {
		border-color: var(--danger);
		color: var(--danger);
	}

	.empty {
		display: grid;
		justify-items: center;
		gap: 0.5rem;
		padding: 3rem 1rem;
		border: 1px dashed var(--border-strong);
		border-radius: var(--radius);
		color: var(--muted);
	}

	.empty p {
		margin: 0;
	}

	.total {
		margin-top: 1rem;
	}
</style>
