<script lang="ts">
	import Editor from '$lib/components/admin/Editor.svelte';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';

	let { form } = $props();

	const empty = { slug: '', title: '', body: '' };
	let values = $derived(form?.values ?? empty);
</script>

<svelte:head>
	<title>New page - nilptr admin</title>
</svelte:head>

<h1>New page</h1>
<FormMessage error={form?.error} />
<Editor {values} autoSlug submitLabel="Create" />
