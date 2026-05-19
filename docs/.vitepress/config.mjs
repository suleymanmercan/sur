import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'sur',
  description: 'Local-first Linux/VPS hardening and setup assistant',
  lang: 'tr-TR',
  base: '/sur/',
  appearance: 'dark',

  themeConfig: {
    siteTitle: 'sur',

    nav: [
      { text: 'Ana Sayfa', link: '/' },
      { text: 'Kurulum', link: '/kurulum' },
      { text: 'Komutlar', link: '/komutlar' },
      { text: 'Güvenlik', link: '/guvenlik' }
    ],

    sidebar: [
      {
        text: 'Dokümantasyon',
        items: [
          { text: 'Kurulum', link: '/kurulum' },
          { text: 'Komutlar', link: '/komutlar' },
          { text: 'Task Sistemi', link: '/task-sistemi' },
          { text: 'Güvenlik Notları', link: '/guvenlik' },
          { text: 'Proje Durumu', link: '/durum' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/suleymanmercan/sur' }
    ]
  }
})
