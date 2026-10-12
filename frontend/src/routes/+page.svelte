<script lang="ts">
	import { resolve } from '$app/paths';
	import Icon from '$lib/components/Icon.svelte';
	import { formatDateLong } from '$lib/format';
	import { SITE_DESCRIPTION, SITE_NAME, SITE_TAGLINE } from '$lib/nav';

	let { data } = $props();

	let first = $derived(data.page === 1);

	const [tagline, taglineAccent = ''] = SITE_TAGLINE.split('|').map((part) => part.trim());
</script>

<svelte:head>
	<title>{first ? SITE_NAME : `Page ${data.page} - ${SITE_NAME}`}</title>
	<meta name="description" content={SITE_DESCRIPTION} />
</svelte:head>

<div class="wrap">
	{#if first}
		<section class="hero">
			<p class="kicker"><span class="pulse"></span> Writing about software</p>
			<h1>{tagline} <span class="accent">{taglineAccent}</span></h1>
			<p class="lead">{SITE_DESCRIPTION}</p>
		</section>
	{/if}

	<div class="section-title">
		<h2>{first ? 'Latest posts' : `Posts, page ${data.page}`}</h2>
		{#if data.total}
			<span class="count">{data.total} {data.total === 1 ? 'post' : 'posts'}</span>
		{/if}
	</div>

	{#if data.posts.length === 0}
		<div class="empty card">
			<Icon name="file" size="2rem" />
			<p>No posts yet. Check back soon.</p>
		</div>
	{:else}
		<ul class="grid">
			{#each data.posts as post, i (post.slug)}
				<li class:featured={first && i === 0}>
					<article class="post-card">
						<div class="eyebrow">
							<time datetime={post.published_at}>{formatDateLong(post.published_at)}</time>
							<span class="dot"></span>
							<span>{post.minutes} min read</span>
						</div>
						<h3>
							<a href={resolve('/posts/[slug]', { slug: post.slug })}>{post.title}</a>
						</h3>
						{#if post.excerpt}
							<p class="excerpt">{post.excerpt}</p>
						{/if}
						<span class="more" aria-hidden="true">
							Read post <Icon name="arrow-right" />
						</span>
					</article>
				</li>
			{/each}
		</ul>
	{/if}

	{#if data.pages > 1}
		<nav class="pager" aria-label="Pagination">
			{#if data.page > 1}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
				<a class="pager-link" href={data.page === 2 ? resolve('/') : `?page=${data.page - 1}`}>
					<Icon name="arrow-left" /> Newer
				</a>
			{:else}
				<span></span>
			{/if}
			<span class="pager-info">Page {data.page} of {data.pages}</span>
			{#if data.page < data.pages}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
				<a class="pager-link" href={`?page=${data.page + 1}`}>
					Older <Icon name="arrow-right" />
				</a>
			{:else}
				<span></span>
			{/if}
		</nav>
	{/if}
</div>

<style>
	.hero {
		padding-block: 1.5rem 3rem;
		max-width: 46rem;
	}

	.kicker {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 1.25rem;
		padding: 0.3rem 0.8rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--muted);
		font-size: 0.8125rem;
		font-weight: 500;
	}

	.pulse {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		background: var(--accent);
		box-shadow: 0 0 0 0.25rem var(--accent-soft);
	}

	.hero h1 {
		margin-bottom: 1rem;
		font-size: clamp(2rem, 5vw, 3.25rem);
		line-height: 1.1;
		letter-spacing: -0.035em;
	}

	.accent {
		color: var(--accent);
	}

	.lead {
		margin: 0;
		color: var(--muted);
		font-size: 1.125rem;
	}

	.section-title {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 1rem;
		margin-bottom: 1.25rem;
	}

	.section-title h2 {
		margin: 0;
		font-size: 1.25rem;
	}

	.count {
		color: var(--muted);
		font-size: 0.875rem;
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 20rem), 1fr));
		gap: 1.25rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.grid > li {
		display: flex;
		margin: 0;
		list-style: none;
	}

	@media (min-width: 768px) {
		.featured {
			grid-column: 1 / -1;
		}

		.featured .post-card {
			padding: 2.25rem;
		}

		.featured h3 {
			font-size: 1.75rem;
		}

		.featured .excerpt {
			max-width: 46rem;
			font-size: 1.0625rem;
			-webkit-line-clamp: 4;
			line-clamp: 4;
		}
	}

	.post-card {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		width: 100%;
		margin: 0;
		padding: 1.5rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
		box-shadow: var(--shadow-sm);
		transition:
			transform 0.2s var(--ease),
			box-shadow 0.2s var(--ease),
			border-color 0.2s var(--ease);
	}

	.post-card:hover {
		transform: translateY(-3px);
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
		box-shadow: var(--shadow-lg);
	}

	.post-card:focus-within {
		border-color: var(--accent);
	}

	.post-card h3 {
		margin: 0;
		font-size: 1.25rem;
		line-height: 1.3;
	}

	.post-card h3 a {
		color: var(--heading);
		text-decoration: none;
	}

	/* The whole card is clickable through the title link. */
	.post-card h3 a::after {
		content: '';
		position: absolute;
		inset: 0;
		border-radius: inherit;
	}

	.post-card h3 a:focus-visible {
		outline: none;
	}

	.excerpt {
		display: -webkit-box;
		margin: 0;
		overflow: hidden;
		color: var(--muted);
		font-size: 0.9375rem;
		line-height: 1.65;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 3;
		line-clamp: 3;
	}

	.more {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		margin-top: auto;
		padding-top: 0.25rem;
		color: var(--accent);
		font-size: 0.875rem;
		font-weight: 600;
	}

	.more :global(svg) {
		transition: transform 0.2s var(--ease);
	}

	.post-card:hover .more :global(svg) {
		transform: translateX(3px);
	}

	.empty {
		display: grid;
		justify-items: center;
		gap: 0.5rem;
		padding: 3rem 1rem;
		color: var(--muted);
		text-align: center;
	}

	.empty p {
		margin: 0;
	}

	.pager {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin-top: 2.5rem;
	}

	.pager-link {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.55rem 1rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--text);
		font-size: 0.9375rem;
		font-weight: 500;
		text-decoration: none;
	}

	.pager-link:hover {
		border-color: var(--accent);
		color: var(--accent);
	}

	.pager-info {
		color: var(--muted);
		font-size: 0.875rem;
	}
</style>
