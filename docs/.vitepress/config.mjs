import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'sur',
  description: 'Local-first Linux/VPS hardening and setup assistant',
  lang: 'tr-TR',
  base: '/sur/',
  appearance: 'dark',

  themeConfig: {
    siteTitle: 'sur',
    outline: {
      level: [2, 3],
      label: 'Bu sayfada'
    },
    search: {
      provider: 'local'
    },
    footer: {
      message: 'Local-first Linux/VPS hardening assistant.',
      copyright: 'Released under the MIT License.'
    },

    nav: [
      { text: 'Ana Sayfa', link: '/' },
      { text: 'Kurulum', link: '/kurulum' },
      { text: 'Komutlar', link: '/komutlar' },
      {
        text: 'Task Yazma',
        items: [
          { text: 'Task Sistemi', link: '/task-sistemi' },
          { text: 'YAML Rehberi', link: '/yaml-rehberi' },
          { text: 'Lua Rehberi', link: '/lua-rehberi' }
        ]
      },
      { text: 'Güvenlik', link: '/guvenlik' }
    ],

    sidebar: [
      {
        text: 'Başlangıç',
        items: [
          { text: 'Kurulum', link: '/kurulum' },
          { text: 'Komutlar', link: '/komutlar' },
          { text: 'Güvenlik Notları', link: '/guvenlik' },
          { text: 'Proje Durumu', link: '/durum' }
        ]
      },
      {
        text: 'Task Yazma',
        items: [
          { text: 'Task Sistemi', link: '/task-sistemi' },
          { text: 'YAML Rehberi', link: '/yaml-rehberi' },
          { text: 'Lua Rehberi', link: '/lua-rehberi' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/suleymanmercan/sur' }
    ]
  }
})
