<script lang="ts">
    import '@svelteness/kit-docs/client/polyfills/index.js';
    import '@svelteness/kit-docs/client/styles/vars.css';

    import { rootUrl } from "$lib/index"

    import {
      KitDocs,
      KitDocsLayout,
      createSidebarContext,
	    Button,
    } from '@svelteness/kit-docs';
  
    export let data;
  
    let { meta, sidebar } = data;
    $: ({ meta, sidebar } = data);
  
    const { activeCategory } = createSidebarContext(sidebar);
  
    $: category = $activeCategory ? `${$activeCategory}: ` : '';
    $: title = meta ? `${category}${meta.title} | Svelte` : null;
    $: description = meta?.description;

    const image = "/guide/static/favicon.ico?l=1"
</script>

<svelte:head>
  <title>{title}</title>
  <meta name="msapplication-TileColor" content="#220A49" />
  <meta name="theme-color" content="#220A49" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <meta http-equiv="Content-Language" content="en" />
  <meta name="description" content={description} />
  <meta name="og:description" content={description} />
  <meta name="og:title" content={title} />
  <meta name="og:image" content={image} />
  <meta name="twitter:card" content="summary" />
  <meta name="twitter:site" content="Infinity Bots" />
  <meta name="twitter:creator" content="@InfinityBotList" />
  <meta property="og:site_name" content="Infinity Docs" />
  <meta name="apple-mobile-web-app-title" content={title} />
  <link rel="apple-touch-icon" sizes="180x180" href={image} />
  <link rel="manifest" href="/guide/static/manifest.json" />
  <link rel="apple-touch-icon" href="/guide/static/pwa_logo.png" />
  <meta name="theme-color" content="#220A49" />
  <link rel="icon" sizes="192x192" href={image} />
  <link rel="icon" sizes="32x32" href={image} />
  <link rel="icon" sizes="96x96" href={image} />
  <link rel="icon" sizes="16x16" href={image} />
  <link rel="icon" href={image} type="image/x-icon" />
  <meta name="msapplication-TileImage" content={image} />  
</svelte:head>
    
<KitDocs {meta}>
  <KitDocsLayout navbar={{
    links: [
      { slug: rootUrl, title: "Main Site" },
    ]
  }} {sidebar}>  
    <div slot="navbar-left">
      <div class="logo">
        <Button href={rootUrl}>
          <img src="https://cdn.infinitybots.gg/images/png/Infinity.png" class="mr-3 h-6 sm:h-9" alt="IBL Logo" />
          <span class="self-center text-xl font-semibold whitespace-nowrap dark:text-white">Infinity Docs</span>
        </Button>
      </div>
    </div>
    <slot />
  </KitDocsLayout>
</KitDocs>