<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';
	import Pager from '$lib/components/admin/Pager.svelte';
	import { formatDateTime } from '$lib/format';

	let { data, form } = $props();
</script>

<svelte:head>
	<title>Comments - nilptr admin</title>
</svelte:head>

<h1>Comments</h1>

<nav aria-label="Comment status">
	<ul>
		{#each data.statuses as status (status)}
			<li>
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
				<a href="?status={status}" aria-current={status === data.status ? 'page' : undefined}>
					{status}
				</a>
			</li>
		{/each}
	</ul>
</nav>

<FormMessage error={form?.error} success={form && 'success' in form ? form.success : null} />

{#if data.comments.length === 0}
	<p>No {data.status} comments.</p>
{:else}
	{#each data.comments as comment (comment.id)}
		<article>
			<header>
				<strong>{comment.author || 'Anonymous'}</strong>
				on <a href={resolve('/posts/[slug]', { slug: comment.post_slug })}>{comment.post_title}</a>
				<br />
				<small>{formatDateTime(comment.created_at)}, {comment.status}</small>
			</header>
			<p class="body">{comment.body}</p>
			<footer>
				<form method="POST" use:enhance class="actions">
					<input type="hidden" name="id" value={comment.id} />
					{#if comment.status !== 'approved'}
						<button type="submit" formaction="?/approve">Approve</button>
					{/if}
					{#if comment.status !== 'rejected'}
						<button type="submit" formaction="?/reject" class="secondary">Reject</button>
					{/if}
					<button type="submit" formaction="?/delete" class="secondary outline">Delete</button>
				</form>
			</footer>
		</article>
	{/each}
	<p><small>{data.total} {data.status} comments</small></p>
{/if}

<Pager page={data.page} pages={data.pages} />

<style>
	nav a[aria-current='page'] {
		font-weight: bold;
		text-decoration: underline;
	}

	.body {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.actions {
		display: flex;
		gap: 0.5rem;
		margin: 0;
	}

	.actions button {
		margin: 0;
	}
</style>
