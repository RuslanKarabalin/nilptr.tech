<script lang="ts">
	import CommentForm from '$lib/components/CommentForm.svelte';
	import Markdown from '$lib/components/Markdown.svelte';
	import { formatDate, formatDateTime } from '$lib/format';
	import { SITE_NAME } from '$lib/nav';

	let { data, form } = $props();
</script>

<svelte:head>
	<title>{data.post.title} - {SITE_NAME}</title>
	{#if data.description}
		<meta name="description" content={data.description} />
	{/if}
	{#if data.post.status !== 'published'}
		<meta name="robots" content="noindex" />
	{/if}
</svelte:head>

<article>
	<header>
		<h1>{data.post.title}</h1>
		{#if data.post.published_at}
			<small>
				<time datetime={data.post.published_at}>{formatDate(data.post.published_at)}</time>
			</small>
		{/if}
	</header>

	<Markdown html={data.html} />
</article>

<section class="comments" aria-labelledby="comments-title">
	<h2 id="comments-title">Comments</h2>

	{#if data.comments === null}
		<p><small>Comments could not be loaded.</small></p>
	{:else if data.comments.length === 0}
		<p><small>No comments yet.</small></p>
	{:else}
		{#each data.comments as comment (comment.id)}
			<div class="comment">
				<p class="meta">
					<strong>{comment.author || 'Anonymous'}</strong>
					<small
						><time datetime={comment.created_at}>{formatDateTime(comment.created_at)}</time></small
					>
				</p>
				<p class="body">{comment.body}</p>
			</div>
		{/each}
	{/if}

	<h3>Leave a comment</h3>
	<CommentForm slug={data.post.slug} fallback={form?.comment ?? null} />
</section>

<style>
	.comments {
		margin-top: 3rem;
	}

	.comment {
		padding-block: 0.75rem;
		border-bottom: 1px solid var(--pico-muted-border-color);
	}

	.comment p {
		margin-bottom: 0.25rem;
	}

	.meta {
		display: flex;
		gap: 0.75rem;
		align-items: baseline;
	}

	.body {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
</style>
