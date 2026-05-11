<script lang="ts">
	let count = $state(0);

	let inc = () => {
		++count;
	};

	let dec = () => {
		++count;
	};
</script>

<div style="display: flex; margin: 1rem 0;">
	<button onclick={dec}>-</button>
	<h1 style="margin: 1rem;">{count}</h1>
	<button onclick={inc}>+</button>
</div>
