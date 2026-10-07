<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatDateTime } from '$lib/format';

	let { data } = $props();
</script>

<svelte:head>
	<title>Dashboard - nilptr admin</title>
</svelte:head>

<h1>Dashboard</h1>

<div class="grid cards">
	<article>
		<header>Posts</header>
		<p><strong>{data.counts.posts}</strong> total, {data.counts.drafts} drafts</p>
		<a href={resolve('/admin/posts/new')} role="button">New post</a>
	</article>
	<article>
		<header>Pages</header>
		<p><strong>{data.counts.pages}</strong> total</p>
		<a href={resolve('/admin/pages/new')} role="button" class="secondary">New page</a>
	</article>
	<article>
		<header>Comments</header>
		<p><strong>{data.counts.pending}</strong> waiting for moderation</p>
		<a href={resolve('/admin/comments')} role="button" class="secondary">Moderate</a>
	</article>
	<article>
		<header>Files</header>
		<p><strong>{data.counts.files}</strong> uploaded</p>
		<a href={resolve('/admin/files')} role="button" class="secondary">Upload</a>
	</article>
</div>

<h2>Recently updated posts</h2>
{#if data.recent.length === 0}
	<p>No posts yet.</p>
{:else}
	<table>
		<thead>
			<tr><th>Title</th><th>Status</th><th>Updated</th></tr>
		</thead>
		<tbody>
			{#each data.recent as post (post.id)}
				<tr>
					<td><a href={resolve('/admin/posts/[id]', { id: post.id })}>{post.title}</a></td>
					<td>{post.status}</td>
					<td>{formatDateTime(post.updated_at)}</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}
