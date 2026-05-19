import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "sur",
  description: "Linux hardening, with consent",
  
  // Varsayılan olarak hep koyu tema olmasını istiyorsak:
  appearance: 'dark',
  
  // Eğer /docs altındaysa, projenin root yolunu değiştirmene genelde gerek yok.
  // Sadece GitHub Pages'da repo adınla yayınlanacaksa base eklemek gerekebilir.
  // base: '/sur/', 
  
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Ana Sayfa', link: '/' },
      { text: 'Kurulum', link: '/kurulum' }
    ],

    sidebar: [
      {
        text: 'Dokümantasyon',
        items: [
          { text: 'Kurulum', link: '/kurulum' },
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/suleymanmercan/sur' }
    ]
  }
})
