<script lang="ts">
	import SiteHeader from '$lib/components/SiteHeader.svelte';
	import GoTopButton from '$lib/components/GoTopButton.svelte';

	let { children } = $props();
</script>

<div class="page">
	<SiteHeader />

	<main class="container">
		{@render children()}
	</main>

	<GoTopButton />
</div>

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
