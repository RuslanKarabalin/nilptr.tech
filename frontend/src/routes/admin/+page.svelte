<script lang="ts">
	import { resolve } from '$app/paths';
	import Icon from '$lib/components/Icon.svelte';
	import { formatDateTime } from '$lib/format';

	let { data } = $props();

	let tiles = $derived([
		{
			label: 'Posts',
			value: data.counts.posts,
			note: `${data.counts.drafts} ${data.counts.drafts === 1 ? 'draft' : 'drafts'}`,
			icon: 'file',
			path: '/admin/posts'
		},
		{
			label: 'Pages',
			value: data.counts.pages,
			note: 'Standalone pages',
			icon: 'pages',
			path: '/admin/pages'
		},
		{
			label: 'Pending comments',
			value: data.counts.pending,
			note: data.counts.pending ? 'Waiting for moderation' : 'All caught up',
			icon: 'message',
			path: '/admin/comments',
			alert: data.counts.pending > 0
		},
		{
			label: 'Files',
			value: data.counts.files,
			note: 'Uploaded to storage',
			icon: 'folder',
			path: '/admin/files'
		}
	] as const);
</script>

<svelte:head>
	<title>Dashboard - nilptr admin</title>
</svelte:head>

<div class="heading">
	<h1>Dashboard</h1>
	<a href={resolve('/admin/posts/new')} role="button"><Icon name="plus" /> New post</a>
</div>

<div class="tiles">
	{#each tiles as tile (tile.label)}
		<a class="tile" class:alert={'alert' in tile && tile.alert} href={resolve(tile.path)}>
			<span class="tile-icon"><Icon name={tile.icon} /></span>
			<span class="tile-value">{tile.value}</span>
			<span class="tile-label">{tile.label}</span>
			<span class="tile-note">{tile.note}</span>
		</a>
	{/each}
</div>

<div class="section-head">
	<h2>Recently updated posts</h2>
	<a href={resolve('/admin/posts')}>All posts <Icon name="arrow-right" /></a>
</div>

{#if data.recent.length === 0}
	<p class="empty">No posts yet.</p>
{:else}
	<div class="overflow-auto">
		<table>
			<thead>
				<tr><th>Title</th><th>Status</th><th>Updated</th></tr>
			</thead>
			<tbody>
				{#each data.recent as post (post.id)}
					<tr>
						<td><a href={resolve('/admin/posts/[id]', { id: post.id })}>{post.title}</a></td>
						<td><span class="badge badge-{post.status}">{post.status}</span></td>
						<td class="muted">{formatDateTime(post.updated_at)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 10rem), 1fr));
		gap: 1rem;
	}

	.tile {
		display: grid;
		grid-template-columns: 1fr auto;
		grid-template-areas:
			'value icon'
			'label label'
			'note note';
		gap: 0.1rem;
		padding: 1.25rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		box-shadow: var(--shadow-sm);
		color: inherit;
		text-decoration: none;
		transition:
			transform 0.2s var(--ease),
			box-shadow 0.2s var(--ease),
			border-color 0.2s var(--ease);
	}

	.tile:hover {
		transform: translateY(-2px);
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
		box-shadow: var(--shadow);
	}

	.tile-icon {
		grid-area: icon;
		display: grid;
		place-items: center;
		width: 2.5rem;
		height: 2.5rem;
		border-radius: var(--radius-sm);
		background: var(--accent-soft);
		color: var(--accent);
	}

	.tile.alert .tile-icon {
		background: var(--warning-soft);
		color: var(--warning);
	}

	.tile-value {
		grid-area: value;
		color: var(--heading);
		font-size: 2rem;
		font-weight: 700;
		line-height: 1.1;
		letter-spacing: -0.03em;
		font-variant-numeric: tabular-nums;
	}

	.tile-label {
		grid-area: label;
		margin-top: 0.4rem;
		color: var(--heading);
		font-weight: 600;
	}

	.tile-note {
		grid-area: note;
		color: var(--muted);
		font-size: 0.8125rem;
	}

	.section-head {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 1rem;
		margin: 2.5rem 0 1rem;
	}

	.section-head h2 {
		margin: 0;
	}

	.section-head a {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
	}

	.muted,
	.empty {
		color: var(--muted);
	}
</style>
