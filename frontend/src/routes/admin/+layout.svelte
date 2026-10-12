<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Brand from '$lib/components/Brand.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';

	let { data, children } = $props();

	const links = [
		{ path: '/admin', label: 'Dashboard', icon: 'gauge' },
		{ path: '/admin/posts', label: 'Posts', icon: 'file' },
		{ path: '/admin/pages', label: 'Pages', icon: 'pages' },
		{ path: '/admin/nav', label: 'Navigation', icon: 'menu' },
		{ path: '/admin/files', label: 'Files', icon: 'folder' },
		{ path: '/admin/comments', label: 'Comments', icon: 'message' },
		{ path: '/admin/stats', label: 'Stats', icon: 'chart' },
		{ path: '/admin/sessions', label: 'Sessions', icon: 'shield' }
	] as const;

	function isCurrent(target: string): boolean {
		const path = page.url.pathname;
		if (target === '/admin') return path === target;
		return path === target || path.startsWith(target + '/');
	}
</script>

<svelte:head>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

{#if data.user}
	<div class="admin">
		<aside class="sidebar">
			<div class="sidebar-top">
				<Brand admin />
				<div class="mobile-tools">
					<a class="icon-button" href={resolve('/')} aria-label="View site" title="View site">
						<Icon name="globe" />
					</a>
					<ThemeToggle />
					<form method="POST" action="/admin/logout">
						<button type="submit" class="icon-button" aria-label="Log out" title="Log out">
							<Icon name="logout" />
						</button>
					</form>
				</div>
			</div>

			<nav aria-label="Admin">
				{#each links as link (link.path)}
					<a href={resolve(link.path)} aria-current={isCurrent(link.path) ? 'page' : undefined}>
						<Icon name={link.icon} />
						<span>{link.label}</span>
					</a>
				{/each}
			</nav>

			<div class="sidebar-bottom">
				<a class="site-link" href={resolve('/')}>
					<Icon name="globe" />
					<span>View site</span>
				</a>
				<div class="user">
					<span class="avatar" aria-hidden="true">{data.user.login[0]?.toUpperCase()}</span>
					<span class="login">{data.user.login}</span>
					<ThemeToggle />
					<form method="POST" action="/admin/logout">
						<button type="submit" class="icon-button" aria-label="Log out" title="Log out">
							<Icon name="logout" />
						</button>
					</form>
				</div>
			</div>
		</aside>

		<main class="admin-main">
			{@render children()}
		</main>
	</div>
{:else}
	<main class="admin-guest">
		{@render children()}
	</main>
{/if}

<style>
	.admin {
		display: grid;
		grid-template-columns: 15.5rem minmax(0, 1fr);
		min-height: 100dvh;
	}

	.sidebar {
		position: sticky;
		top: 0;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		height: 100dvh;
		padding: 1.25rem 0.9rem;
		box-sizing: border-box;
		border-right: 1px solid var(--border);
		background: var(--surface);
	}

	.sidebar-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding-inline: 0.4rem;
	}

	.mobile-tools {
		display: none;
	}

	nav {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}

	nav a,
	.site-link {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		padding: 0.55rem 0.75rem;
		border-radius: var(--radius-sm);
		color: var(--muted);
		font-size: 0.9375rem;
		font-weight: 500;
		text-decoration: none;
		white-space: nowrap;
		transition:
			background 0.15s var(--ease),
			color 0.15s var(--ease);
	}

	nav a:hover,
	.site-link:hover {
		background: var(--surface-2);
		color: var(--heading);
	}

	nav a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--accent);
		font-weight: 600;
	}

	nav a :global(svg),
	.site-link :global(svg) {
		width: 1.1rem;
		height: 1.1rem;
		flex: none;
	}

	.sidebar-bottom {
		display: grid;
		gap: 0.5rem;
		margin-top: auto;
	}

	.user {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.6rem 0.5rem 0;
		border-top: 1px solid var(--border);
	}

	.user form,
	.mobile-tools form {
		margin: 0;
	}

	.avatar {
		display: grid;
		place-items: center;
		width: 2rem;
		height: 2rem;
		flex: none;
		border-radius: 50%;
		background: var(--accent-soft);
		color: var(--accent);
		font-size: 0.875rem;
		font-weight: 700;
	}

	.login {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		color: var(--heading);
		font-size: 0.9375rem;
		font-weight: 600;
		text-overflow: ellipsis;
	}

	.admin-main {
		width: 100%;
		max-width: 76rem;
		padding: 2.25rem clamp(1rem, 3vw, 2.5rem) 4rem;
		box-sizing: border-box;
	}

	.admin-guest {
		display: grid;
		place-items: center;
		min-height: 100dvh;
		padding: 1rem;
		box-sizing: border-box;
	}

	@media (max-width: 900px) {
		.admin {
			grid-template-columns: minmax(0, 1fr);
			/* Without this a short page stretches both rows to fill the
			   screen and the top bar grows. */
			grid-template-rows: auto 1fr;
		}

		.sidebar {
			position: sticky;
			z-index: 10;
			height: auto;
			gap: 0.75rem;
			padding: 0.75rem var(--gutter) 0;
			border-right: 0;
			border-bottom: 1px solid var(--border);
			background: var(--header-bg);
			backdrop-filter: saturate(1.6) blur(14px);
			-webkit-backdrop-filter: saturate(1.6) blur(14px);
		}

		.sidebar-top {
			padding: 0;
		}

		.mobile-tools {
			display: flex;
			align-items: center;
			gap: 0.15rem;
		}

		nav {
			flex-direction: row;
			margin-inline: calc(-1 * var(--gutter));
			padding: 0 var(--gutter) 0.6rem;
			overflow-x: auto;
			scrollbar-width: none;
		}

		nav::-webkit-scrollbar {
			display: none;
		}

		nav a {
			padding: 0.45rem 0.7rem;
			font-size: 0.875rem;
		}

		.sidebar-bottom {
			display: none;
		}
	}

	/* Shared look of admin pages. */

	.admin-main :global(h1) {
		margin-bottom: 1.5rem;
		font-size: 1.75rem;
	}

	.admin-main :global(h2) {
		margin-top: 2.5rem;
		font-size: 1.2rem;
	}

	.admin-main :global(.heading) {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.admin-main :global(.heading h1) {
		margin: 0;
	}

	.admin-main :global(.heading [role='button']) {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		margin: 0;
	}

	.admin-main :global(table) {
		margin: 0;
		border-collapse: separate;
		border-spacing: 0;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		box-shadow: var(--shadow-sm);
		overflow: hidden;
	}

	.admin-main :global(.overflow-auto) {
		margin-bottom: 1rem;
		border-radius: var(--radius);
	}

	.admin-main :global(th),
	.admin-main :global(td) {
		padding: 0.75rem 1rem;
		border-bottom: 1px solid var(--border);
		background: transparent;
		vertical-align: middle;
	}

	.admin-main :global(thead th) {
		background: var(--surface-2);
	}

	.admin-main :global(tbody tr:last-child td) {
		border-bottom: 0;
	}

	.admin-main :global(tbody tr) {
		transition: background 0.1s var(--ease);
	}

	.admin-main :global(tbody tr:hover td) {
		background: color-mix(in srgb, var(--surface-2) 60%, transparent);
	}

	/* Dates, sizes, statuses and actions stay on one line; the first
	   column (title, name, device) wraps but never gets squeezed. */
	.admin-main :global(td) {
		white-space: nowrap;
	}

	.admin-main :global(td:first-child) {
		min-width: 11rem;
		white-space: normal;
		overflow-wrap: anywhere;
	}

	.admin-main :global(td a) {
		font-weight: 500;
		text-decoration: none;
	}

	.admin-main :global(td a:hover) {
		text-decoration: underline;
	}

	.admin-main :global(article) {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		box-shadow: var(--shadow-sm);
	}
</style>
