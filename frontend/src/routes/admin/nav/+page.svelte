<script lang="ts">
	import { enhance } from '$app/forms';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';
	import type { NavLink } from '$lib/types';

	let { data, form } = $props();

	type Location = 'header' | 'footer';
	type Row = NavLink & { key: number };

	let nextKey = 0;
	const rows = (items: NavLink[]): Row[] =>
		items.map((item) => ({ label: item.label, url: item.url, key: nextKey++ }));

	let source = $derived(form && 'nav' in form && form.nav ? form.nav : data.nav);
	let header = $derived(rows(source.header));
	let footer = $derived(rows(source.footer));
	let saving = $state(false);

	function list(location: Location): Row[] {
		return location === 'header' ? header : footer;
	}

	function update(location: Location, items: Row[]) {
		if (location === 'header') header = items;
		else footer = items;
	}

	function add(location: Location) {
		update(location, [...list(location), { label: '', url: '', key: nextKey++ }]);
	}

	function remove(location: Location, index: number) {
		update(
			location,
			list(location).filter((_, i) => i !== index)
		);
	}

	function edit(location: Location, index: number, key: 'label' | 'url', value: string) {
		update(
			location,
			list(location).map((item, i) => (i === index ? { ...item, [key]: value } : item))
		);
	}

	function move(location: Location, index: number, delta: number) {
		const items = [...list(location)];
		const target = index + delta;
		if (target < 0 || target >= items.length) return;
		[items[index], items[target]] = [items[target], items[index]];
		update(location, items);
	}
</script>

<svelte:head>
	<title>Navigation - nilptr admin</title>
</svelte:head>

<h1>Navigation</h1>
<p>
	<small>
		Links starting with <code>/</code> are internal (for example <code>/cv</code>), anything else
		opens in a new tab.
	</small>
</p>

<FormMessage error={form?.error} success={form && 'success' in form ? form.success : null} />

<form
	method="POST"
	action="?/save"
	use:enhance={() => {
		saving = true;
		return async ({ update }) => {
			await update({ reset: false });
			saving = false;
		};
	}}
>
	{#each ['header', 'footer'] as const as location (location)}
		{@const items = list(location)}
		<section>
			<h2>{location === 'header' ? 'Header' : 'Footer'}</h2>
			{#each items as item, i (item.key)}
				<div class="row" role="group">
					<input
						name="{location}_label"
						type="text"
						placeholder="Label"
						aria-label="Label"
						maxlength="100"
						value={item.label}
						oninput={(e) => edit(location, i, 'label', e.currentTarget.value)}
					/>
					<input
						name="{location}_url"
						type="text"
						placeholder="/path or https://..."
						aria-label="URL"
						maxlength="2000"
						value={item.url}
						oninput={(e) => edit(location, i, 'url', e.currentTarget.value)}
					/>
					<button
						type="button"
						class="secondary outline"
						title="Move up"
						aria-label="Move up"
						disabled={i === 0}
						onclick={() => move(location, i, -1)}>Up</button
					>
					<button
						type="button"
						class="secondary outline"
						title="Move down"
						aria-label="Move down"
						disabled={i === items.length - 1}
						onclick={() => move(location, i, 1)}>Down</button
					>
					<button type="button" class="secondary" onclick={() => remove(location, i)}>Remove</button
					>
				</div>
			{/each}
			<noscript>
				<div class="row" role="group">
					<input name="{location}_label" type="text" placeholder="New label" aria-label="Label" />
					<input name="{location}_url" type="text" placeholder="New URL" aria-label="URL" />
				</div>
			</noscript>
			<button type="button" class="outline" onclick={() => add(location)}
				>Add {location} link</button
			>
		</section>
	{/each}

	<hr />
	<button type="submit" aria-busy={saving} disabled={saving}>Save navigation</button>
</form>

<style>
	section {
		margin-bottom: 2rem;
	}

	.row button {
		flex: 0 0 auto;
	}
</style>
