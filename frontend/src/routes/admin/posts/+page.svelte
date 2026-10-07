<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';
	import Pager from '$lib/components/admin/Pager.svelte';
	import { formatDateTime } from '$lib/format';
	import { POST_STATUSES } from '$lib/types';

	let { data, form } = $props();
</script>

<svelte:head>
	<title>Posts - nilptr admin</title>
</svelte:head>

<div class="heading">
	<h1>Posts</h1>
	<a href={resolve('/admin/posts/new')} role="button">New post</a>
</div>

<form method="GET" class="filter">
	<select
		name="status"
		aria-label="Status"
		value={data.status}
		onchange={(e) => e.currentTarget.form?.requestSubmit()}
	>
		<option value="">All statuses</option>
		{#each POST_STATUSES as status (status)}
			<option value={status}>{status}</option>
		{/each}
	</select>
	<noscript><button type="submit">Filter</button></noscript>
</form>

<FormMessage error={form?.error} success={form?.success} />

{#if data.posts.length === 0}
	<p>No posts.</p>
{:else}
	<div class="overflow-auto">
		<table>
			<thead>
				<tr>
					<th>Title</th>
					<th>Slug</th>
					<th>Status</th>
					<th>Updated</th>
					<th>Published</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each data.posts as post (post.id)}
					<tr>
						<td><a href={resolve('/admin/posts/[id]', { id: post.id })}>{post.title}</a></td>
						<td>
							{#if post.status === 'draft'}
								<code>{post.slug}</code>
							{:else}
								<a href={resolve('/posts/[slug]', { slug: post.slug })}><code>{post.slug}</code></a>
							{/if}
						</td>
						<td>{post.status}</td>
						<td>{formatDateTime(post.updated_at)}</td>
						<td>{formatDateTime(post.published_at)}</td>
						<td>
							<form
								method="POST"
								action="?/delete"
								use:enhance={({ cancel }) => {
									if (!confirm(`Delete post "${post.title}"?`)) cancel();
								}}
							>
								<input type="hidden" name="id" value={post.id} />
								<button type="submit" class="outline secondary small">Delete</button>
							</form>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<p><small>{data.total} posts</small></p>
{/if}

<Pager page={data.page} pages={data.pages} />

<style>
	.heading {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
	}

	.filter {
		max-width: 16rem;
	}

	td form {
		margin: 0;
	}

	.small {
		margin: 0;
		padding: 0.125rem 0.5rem;
		font-size: 0.875rem;
	}
</style>
