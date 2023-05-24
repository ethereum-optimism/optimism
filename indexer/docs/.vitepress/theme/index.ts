// import SvgImage from './components/SvgImage.vue'
import defaultTheme from 'vitepress/theme'
import './index.css'

export default {
  ...defaultTheme,
  enhanceApp({ app }) {
    // this is how we would add a custom image
    // app.component('SvgImage', SvgImage)
  },
}
