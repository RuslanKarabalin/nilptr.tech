<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Pathname } from '$app/types';
	import { isInternalUrl } from '$lib/format';
	import type { NavLink } from '$lib/types';
	import Icon, { iconForUrl } from './Icon.svelte';

	let { link, current = false }: { link: NavLink; current?: boolean } = $props();
</script>

{#if isInternalUrl(link.url)}
	<a href={resolve(link.url as Pathname)} aria-current={current ? 'page' : undefined}
		>{link.label}</a
	>
{:else}
	<a href={link.url} target="_blank" rel="noopener noreferrer external" title={link.label}>
		<Icon name={iconForUrl(link.url)} />
		<span class="link-label-external">{link.label}</span>
	</a>
{/if}
