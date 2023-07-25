import { sveltekit } from '@sveltejs/kit/vite';
import icons from 'unplugin-icons/vite';
import kitDocs from '@svelteness/kit-docs/node';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [icons({ compiler: 'svelte' }), kitDocs(), sveltekit()],
});