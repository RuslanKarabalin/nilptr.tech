<script lang="ts">
	import { enhance } from '$app/forms';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';
	import Pager from '$lib/components/admin/Pager.svelte';
	import { directiveFor, formatDateTime, formatSize } from '$lib/format';

	let { data, form } = $props();

	let uploading = $state(false);
	let copied = $state<string | null>(null);

	async function copy(text: string, id: string) {
		try {
			await navigator.clipboard.writeText(text);
			copied = id;
			setTimeout(() => {
				if (copied === id) copied = null;
			}, 2000);
		} catch {
			copied = null;
		}
	}
</script>

<svelte:head>
	<title>Files - nilptr admin</title>
</svelte:head>

<h1>Files</h1>

<form
	method="POST"
	action="?/upload"
	enctype="multipart/form-data"
	use:enhance={() => {
		uploading = true;
		return async ({ update }) => {
			await update();
			uploading = false;
		};
	}}
>
	<div role="group">
		<input type="file" name="file" required aria-label="File" />
		<button type="submit" aria-busy={uploading} disabled={uploading}>Upload</button>
	</div>
</form>

<FormMessage error={form?.error} success={form && 'success' in form ? form.success : null} />

{#if form && 'uploaded' in form && form.uploaded}
	<p>
		Insert into a post: <code>{directiveFor(form.uploaded)}</code>
	</p>
{/if}

{#if data.files.length === 0}
	<p>No files yet.</p>
{:else}
	<div class="overflow-auto">
		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Type</th>
					<th>Size</th>
					<th>Uploaded</th>
					<th>Directive</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each data.files as file (file.id)}
					{@const snippet = directiveFor(file)}
					<tr>
						<td><a href={file.url} target="_blank" rel="external noopener">{file.name}</a></td>
						<td><small>{file.content_type}</small></td>
						<td>{formatSize(file.size)}</td>
						<td><small>{formatDateTime(file.created_at)}</small></td>
						<td class="snippet">
							<code>{snippet}</code>
							<button type="button" class="outline small" onclick={() => copy(snippet, file.id)}>
								{copied === file.id ? 'Copied' : 'Copy'}
							</button>
						</td>
						<td>
							<form
								method="POST"
								action="?/delete"
								use:enhance={({ cancel }) => {
									if (!confirm(`Delete ${file.name}? Links to it will stop working.`)) cancel();
								}}
							>
								<input type="hidden" name="id" value={file.id} />
								<button type="submit" class="outline secondary small">Delete</button>
							</form>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<p><small>{data.total} files</small></p>
{/if}

<Pager page={data.page} pages={data.pages} />

<style>
	td form {
		margin: 0;
	}

	.snippet code {
		white-space: nowrap;
		font-size: 0.75rem;
	}

	.small {
		margin: 0;
		padding: 0.125rem 0.5rem;
		font-size: 0.875rem;
	}
</style>
