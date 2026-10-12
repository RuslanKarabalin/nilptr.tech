<script lang="ts">
	import { enhance } from '$app/forms';
	import Icon from '$lib/components/Icon.svelte';
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
	<div class="dropzone">
		<span class="dropzone-icon"><Icon name="upload" size="1.4rem" /></span>
		<div class="dropzone-text">
			<strong>Upload a file</strong>
			<span>Images, video or any other file. It gets a permanent link.</span>
		</div>
		<input type="file" name="file" required aria-label="File" />
		<button type="submit" aria-busy={uploading} disabled={uploading}>Upload</button>
	</div>
</form>

<FormMessage error={form?.error} success={form && 'success' in form ? form.success : null} />

{#if form && 'uploaded' in form && form.uploaded}
	<p class="insert">
		Insert into a post: <code>{directiveFor(form.uploaded)}</code>
	</p>
{/if}

{#if data.files.length === 0}
	<p class="empty">No files yet.</p>
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

	/* The directive and its Copy button may wrap onto two lines. */
	td.snippet {
		white-space: normal;
	}

	.snippet {
		min-width: 14rem;
	}

	.snippet code {
		display: inline-block;
		margin-bottom: 0.35rem;
		overflow-wrap: anywhere;
		font-size: 0.75rem;
	}

	.small {
		margin: 0;
		padding: 0.3rem 0.7rem;
		font-size: 0.8125rem;
	}

	.small:hover {
		border-color: var(--danger);
		color: var(--danger);
	}

	.dropzone {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		align-items: center;
		gap: 1rem 1.1rem;
		margin-bottom: 1.5rem;
		padding: 1.25rem;
		border: 1.5px dashed var(--border-strong);
		border-radius: var(--radius);
		background: var(--surface);
	}

	.dropzone-icon {
		display: grid;
		place-items: center;
		width: 3rem;
		height: 3rem;
		border-radius: var(--radius-sm);
		background: var(--accent-soft);
		color: var(--accent);
	}

	.dropzone-text {
		display: grid;
		gap: 0.1rem;
	}

	.dropzone-text strong {
		color: var(--heading);
	}

	.dropzone-text span,
	.empty {
		color: var(--muted);
		font-size: 0.9375rem;
	}

	.dropzone input {
		grid-column: 1 / -1;
		margin: 0;
	}

	.dropzone button {
		grid-column: 1 / -1;
		margin: 0;
	}

	@media (min-width: 900px) {
		.dropzone {
			grid-template-columns: auto minmax(0, 1fr) minmax(0, 1.2fr) auto;
		}

		.dropzone input,
		.dropzone button {
			grid-column: auto;
		}
	}

	.insert {
		padding: 0.75rem 1rem;
		border-radius: var(--radius-sm);
		background: var(--accent-soft);
	}
</style>
