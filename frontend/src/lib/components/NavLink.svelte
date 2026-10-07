<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Pathname } from '$app/types';
	import { isInternalUrl } from '$lib/format';
	import type { NavLink } from '$lib/types';

	let { link }: { link: NavLink } = $props();
</script>

{#if isInternalUrl(link.url)}
	<a href={resolve(link.url as Pathname)}>{link.label}</a>
{:else}
	<a href={link.url} target="_blank" rel="noopener noreferrer external">{link.label}</a>
{/if}
