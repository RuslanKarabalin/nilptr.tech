<script lang="ts">
	import { resolve } from '$app/paths';
	import CommentForm from '$lib/components/CommentForm.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import Markdown from '$lib/components/Markdown.svelte';
	import { formatDateLong, formatDateTime } from '$lib/format';
	import { SITE_NAME } from '$lib/nav';

	let { data, form } = $props();

	function initial(name: string | null): string {
		return (name?.trim()[0] ?? '?').toUpperCase();
	}
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

<div class="wrap wrap-prose">
	<a class="back" href={resolve('/')}><Icon name="arrow-left" /> All posts</a>

	<article class="bare">
		<header class="post-header">
			<div class="eyebrow">
				{#if data.post.published_at}
					<time datetime={data.post.published_at}>{formatDateLong(data.post.published_at)}</time>
					<span class="dot"></span>
				{/if}
				<span>{data.minutes} min read</span>
				{#if data.post.status === 'unlisted'}
					<span class="badge badge-unlisted">unlisted</span>
				{/if}
			</div>
			<h1>{data.post.title}</h1>
		</header>

		<div class="post-body card">
			<Markdown html={data.html} />
		</div>
	</article>

	<section class="comments" aria-labelledby="comments-title">
		<h2 id="comments-title">
			<Icon name="message" />
			Comments
			{#if data.comments?.length}<span class="count">{data.comments.length}</span>{/if}
		</h2>

		{#if data.comments === null}
			<p class="note">Comments could not be loaded.</p>
		{:else if data.comments.length === 0}
			<p class="note">No comments yet. Be the first.</p>
		{:else}
			<ol class="comment-list">
				{#each data.comments as comment (comment.id)}
					<li class="comment card">
						<span class="avatar" aria-hidden="true">{initial(comment.author)}</span>
						<div>
							<p class="meta">
								<strong>{comment.author || 'Anonymous'}</strong>
								<time datetime={comment.created_at}>{formatDateTime(comment.created_at)}</time>
							</p>
							<p class="body">{comment.body}</p>
						</div>
					</li>
				{/each}
			</ol>
		{/if}

		<div class="reply card">
			<h3>Leave a comment</h3>
			<CommentForm slug={data.post.slug} fallback={form?.comment ?? null} />
		</div>
	</section>
</div>

<style>
	.back {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		margin-bottom: 1.75rem;
		color: var(--muted);
		font-size: 0.9375rem;
		font-weight: 500;
		text-decoration: none;
	}

	.back:hover {
		color: var(--accent);
	}

	.post-header {
		margin-bottom: 2rem;
	}

	.post-header h1 {
		margin: 0.75rem 0 0;
		font-size: clamp(1.9rem, 4.5vw, 2.75rem);
		line-height: 1.15;
		letter-spacing: -0.035em;
	}

	.post-body {
		padding: clamp(1.25rem, 4vw, 2.75rem);
		border-radius: var(--radius-lg);
	}

	.comments {
		margin-top: 3.5rem;
	}

	.comments h2 {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		margin-bottom: 1.25rem;
		font-size: 1.4rem;
	}

	.comments h2 :global(svg) {
		color: var(--accent);
	}

	.count {
		padding: 0.05rem 0.55rem;
		border-radius: 999px;
		background: var(--accent-soft);
		color: var(--accent);
		font-size: 0.8125rem;
		font-weight: 600;
		letter-spacing: 0;
	}

	.note {
		color: var(--muted);
	}

	.comment-list {
		display: grid;
		gap: 0.75rem;
		margin: 0 0 1.5rem;
		padding: 0;
		list-style: none;
	}

	.comment {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.9rem;
		margin: 0;
		padding: 1rem 1.15rem;
		list-style: none;
	}

	.avatar {
		display: grid;
		place-items: center;
		width: 2.4rem;
		height: 2.4rem;
		border-radius: 50%;
		background: linear-gradient(
			135deg,
			var(--accent-soft),
			color-mix(in srgb, var(--accent) 30%, var(--accent-soft))
		);
		color: var(--accent);
		font-weight: 700;
	}

	.meta {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem 0.75rem;
		align-items: baseline;
		margin: 0 0 0.25rem;
	}

	.meta strong {
		color: var(--heading);
	}

	.meta time {
		color: var(--muted);
		font-size: 0.8125rem;
	}

	.body {
		margin: 0;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.reply {
		padding: clamp(1.25rem, 4vw, 2rem);
		border-radius: var(--radius-lg);
	}

	.reply h3 {
		margin-bottom: 1rem;
		font-size: 1.125rem;
	}
</style>
