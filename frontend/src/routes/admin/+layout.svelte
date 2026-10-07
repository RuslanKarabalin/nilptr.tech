<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';

	let { data, children } = $props();

	const links = [
		{ path: '/admin', label: 'Dashboard' },
		{ path: '/admin/posts', label: 'Posts' },
		{ path: '/admin/pages', label: 'Pages' },
		{ path: '/admin/nav', label: 'Navigation' },
		{ path: '/admin/files', label: 'Files' },
		{ path: '/admin/comments', label: 'Comments' },
		{ path: '/admin/stats', label: 'Stats' },
		{ path: '/admin/sessions', label: 'Sessions' }
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

<header class="admin-header">
	<nav class="container-fluid">
		<ul>
			<li><a href={resolve('/admin')}><strong>nilptr admin</strong></a></li>
		</ul>
		{#if data.user}
			<ul class="links">
				{#each links as link (link.path)}
					<li>
						<a href={resolve(link.path)} aria-current={isCurrent(link.path) ? 'page' : undefined}>
							{link.label}
						</a>
					</li>
				{/each}
			</ul>
		{/if}
		<ul>
			<li><a href={resolve('/')}>Site</a></li>
			<li><ThemeToggle /></li>
			{#if data.user}
				<li>
					<form method="POST" action="/admin/logout" class="logout">
						<button type="submit" class="outline secondary">Log out {data.user.login}</button>
					</form>
				</li>
			{/if}
		</ul>
	</nav>
</header>

<main class="container">
	{@render children()}
</main>

<style>
	.admin-header {
		border-bottom: 1px solid var(--pico-muted-border-color);
	}

	.admin-header nav {
		flex-wrap: wrap;
	}

	.links {
		flex-wrap: wrap;
	}

	.links a[aria-current='page'] {
		text-decoration: underline;
		font-weight: bold;
	}

	.logout {
		margin: 0;
	}

	.logout button {
		margin: 0;
		padding: 0.25rem 0.75rem;
	}

	main {
		padding-block: 2rem;
	}
</style>
