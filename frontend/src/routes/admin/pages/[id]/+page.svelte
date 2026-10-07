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
			: { slug: data.page.slug, title: data.page.title, body: data.page.body }
	);
</script>

<svelte:head>
	<title>Edit {data.page.title} - nilptr admin</title>
</svelte:head>

<div class="heading">
	<h1>Edit page</h1>
	<a href={resolve('/[slug=slug]', { slug: data.page.slug })} target="_blank">Open on site</a>
</div>
<p>
	<small>
		Created {formatDateTime(data.page.created_at)}, updated {formatDateTime(data.page.updated_at)}
	</small>
</p>

<FormMessage
	error={form?.error}
	success={form && 'success' in form ? form.success : data.created ? 'Page created.' : null}
/>

<Editor {values} />

<form
	method="POST"
	action="?/delete"
	use:enhance={({ cancel }) => {
		if (!confirm('Delete this page?')) cancel();
	}}
>
	<button type="submit" class="secondary outline">Delete page</button>
</form>

<style>
	.heading {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 1rem;
	}
</style>
