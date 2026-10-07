<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Editor from '$lib/components/admin/Editor.svelte';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';
	import { formatDateTime } from '$lib/format';

	let { data, form } = $props();

	let values = $derived(
		form && 'values' in form && form.values
			? form.values
			: {
					slug: data.post.slug,
					title: data.post.title,
					body: data.post.body,
					status: data.post.status
				}
	);
</script>

<svelte:head>
	<title>Edit {data.post.title} - nilptr admin</title>
</svelte:head>

<div class="heading">
	<h1>Edit post</h1>
	{#if data.post.status !== 'draft'}
		<a href={resolve('/posts/[slug]', { slug: data.post.slug })} target="_blank">Open on site</a>
	{/if}
</div>
<p>
	<small>
		Created {formatDateTime(data.post.created_at)}, updated {formatDateTime(data.post.updated_at)}
		{#if data.post.published_at}, published {formatDateTime(data.post.published_at)}{/if}
	</small>
</p>

<FormMessage
	error={form?.error}
	success={form && 'success' in form ? form.success : data.created ? 'Post created.' : null}
/>

<Editor {values} withStatus />

<form
	method="POST"
	action="?/delete"
	use:enhance={({ cancel }) => {
		if (!confirm('Delete this post?')) cancel();
	}}
>
	<button type="submit" class="secondary outline">Delete post</button>
</form>

<style>
	.heading {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 1rem;
	}
</style>
