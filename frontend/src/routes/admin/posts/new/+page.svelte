<script lang="ts">
	import Editor from '$lib/components/admin/Editor.svelte';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';

	let { form } = $props();

	const empty = { slug: '', title: '', body: '', status: 'draft' as const };
	let values = $derived(form?.values ?? empty);
</script>

<svelte:head>
	<title>New post - nilptr admin</title>
</svelte:head>

<h1>New post</h1>
<FormMessage error={form?.error} />
<Editor {values} withStatus autoSlug submitLabel="Create" />
