<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatDate } from '$lib/format';
	import { SITE_DESCRIPTION, SITE_NAME } from '$lib/nav';

	let { data } = $props();
</script>

<svelte:head>
	<title>{data.page > 1 ? `Page ${data.page} - ${SITE_NAME}` : SITE_NAME}</title>
	<meta name="description" content={SITE_DESCRIPTION} />
</svelte:head>

<h1>Posts</h1>

{#if data.posts.length === 0}
	<p>No posts yet.</p>
{:else}
	<ul class="feed">
		{#each data.posts as post (post.slug)}
			<li>
				<article>
					<a href={resolve('/posts/[slug]', { slug: post.slug })}>
						<strong>{post.title}</strong>
					</a>
					<br />
					<small><time datetime={post.published_at}>{formatDate(post.published_at)}</time></small>
				</article>
			</li>
		{/each}
	</ul>
{/if}

{#if data.pages > 1}
	<nav aria-label="Pagination">
		<ul>
			{#if data.page > 1}
				<li>
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
					<a href={data.page === 2 ? resolve('/') : `?page=${data.page - 1}`}>Newer</a>
				</li>
			{/if}
		</ul>
		<ul>
			<li><small>Page {data.page} of {data.pages}</small></li>
		</ul>
		<ul>
			{#if data.page < data.pages}
				<li>
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
					<a href={`?page=${data.page + 1}`}>Older</a>
				</li>
			{/if}
		</ul>
	</nav>
{/if}

<style>
	.feed {
		list-style: none;
		padding: 0;
	}

	.feed li {
		list-style: none;
	}

	.feed article {
		margin-bottom: 1rem;
	}
</style>
