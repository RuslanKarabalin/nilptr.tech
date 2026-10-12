<script lang="ts">
	import { resolve } from '$app/paths';
	import { SITE_NAME } from '$lib/nav';

	/** The admin variant links to the dashboard and shows an "admin" tag. */
	let { admin = false }: { admin?: boolean } = $props();
</script>

<a href={admin ? resolve('/admin') : resolve('/')} class="brand">
	<svg class="mark" viewBox="0 0 32 32" aria-hidden="true" focusable="false">
		<rect width="32" height="32" rx="9" />
		<path
			d="M16 7.5v17M8.6 11.75l14.8 8.5M23.4 11.75l-14.8 8.5"
			stroke-width="2.6"
			stroke-linecap="round"
		/>
	</svg>
	<span class="name">{SITE_NAME}</span>
	{#if admin}<span class="suffix">admin</span>{/if}
</a>

<style>
	.brand {
		display: inline-flex;
		align-items: center;
		gap: 0.6rem;
		color: var(--heading);
		font-weight: 700;
		font-size: 1.05rem;
		letter-spacing: -0.02em;
		text-decoration: none;
		white-space: nowrap;
	}

	.mark {
		width: 2rem;
		height: 2rem;
		flex: none;
		transition: transform 0.3s var(--ease);
	}

	.mark rect {
		fill: var(--accent);
	}

	.mark path {
		stroke: var(--accent-contrast);
	}

	.brand:hover .mark {
		transform: rotate(30deg);
	}

	.suffix {
		padding: 0.1rem 0.5rem;
		border-radius: 999px;
		background: var(--accent-soft);
		color: var(--accent);
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0;
	}
</style>
