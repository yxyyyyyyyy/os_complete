import { language, setLanguage, t } from './i18n'

setLanguage('en')
const overviewLabel: string = t.value.nav.overview
const chineseDefault: 'zh' | 'en' = language.value

void overviewLabel
void chineseDefault
