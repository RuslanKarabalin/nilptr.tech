import adapter from '@sveltejs/adapter-static';

const config = {
	compilerOptions: {
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	},
	kit: {
		adapter: adapter({
			fallback: '404.html',
			precompress: false,
			strict: true
		})
	}
};

export default config;
