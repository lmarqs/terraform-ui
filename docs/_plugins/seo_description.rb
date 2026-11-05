# A page's description front matter tells a reader of docs/ what the page is; seo.description, when present, is what
# jekyll-seo-tag puts in the meta tags instead, so the two audiences each get their own sentence.
Jekyll::Hooks.register :pages, :pre_render do |page, payload|
  seo = page.data["seo"]
  payload["page"]["description"] = seo["description"] if seo.is_a?(Hash) && seo["description"]
end
