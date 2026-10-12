<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Icon from '$lib/components/Icon.svelte';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';
	import Pager from '$lib/components/admin/Pager.svelte';
	import { formatDateTime } from '$lib/format';

	let { data, form } = $props();
</script>

<svelte:head>
	<title>Pages - nilptr admin</title>
</svelte:head>

<div class="heading">
	<h1>Pages</h1>
	<a href={resolve('/admin/pages/new')} role="button"><Icon name="plus" /> New page</a>
</div>

<p>
	<small
		>A page with slug <code>about</code> is served at <code>/about</code>. Link it from the header
		or footer in <a href={resolve('/admin/nav')}>Navigation</a>.</small
	>
</p>

<FormMessage error={form?.error} success={form?.success} />

{#if data.pages.length === 0}
	<p>No pages.</p>
{:else}
	<div class="overflow-auto">
		<table>
			<thead>
				<tr><th>Title</th><th>Address</th><th>Updated</th><th></th></tr>
			</thead>
			<tbody>
				{#each data.pages as item (item.id)}
					<tr>
						<td><a href={resolve('/admin/pages/[id]', { id: item.id })}>{item.title}</a></td>
						<td
							><a href={resolve('/[slug=slug]', { slug: item.slug })}><code>/{item.slug}</code></a
							></td
						>
						<td class="muted">{formatDateTime(item.updated_at)}</td>
						<td>
							<form
								method="POST"
								action="?/delete"
								use:enhance={({ cancel }) => {
									if (!confirm(`Delete page "${item.title}"?`)) cancel();
								}}
							>
								<input type="hidden" name="id" value={item.id} />
								<button type="submit" class="outline secondary small">Delete</button>
							</form>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<p><small>{data.total} pages</small></p>
{/if}

<Pager page={data.page} pages={data.pageCount} />

<style>
	td form {
		margin: 0;
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

	.muted {
		color: var(--muted);
		font-size: 0.875rem;
		white-space: nowrap;
	}
</style>
