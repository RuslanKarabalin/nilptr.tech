<script lang="ts">
	import '@picocss/pico/css/pico.min.css';
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import GoTopButton from '$lib/components/GoTopButton.svelte';
	import SiteFooter from '$lib/components/SiteFooter.svelte';
	import SiteHeader from '$lib/components/SiteHeader.svelte';

	let { data, children } = $props();

	const isAdmin = (pathname: string) => pathname === '/admin' || pathname.startsWith('/admin/');

	let admin = $derived(isAdmin(page.url.pathname));

	afterNavigate(({ from, to }) => {
		if (!to) return;
		const { pathname, search } = to.url;
		if (isAdmin(pathname) || page.status >= 400) return;
		// Only the hash changed, it is the same page view.
		if (from && from.url.pathname === pathname && from.url.search === search) return;

		fetch('/api/views', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ path: pathname }),
			keepalive: true
		}).catch(() => {
			// statistics are best effort
		});
	});
</script>

{#if admin}
	{@render children()}
{:else}
	<div class="page">
		<SiteHeader links={data.nav.header} />

		<main class="container">
			{@render children()}
		</main>

		<SiteFooter links={data.nav.footer} />
	</div>
{/if}

<GoTopButton />

<style>
	:global(html, body) {
		margin: 0;
		padding: 0;
	}

	.page {
		min-height: 100dvh;
		display: flex;
		flex-direction: column;
	}

	main {
		flex: 1;
		padding-block: 2rem;
	}
</style>
