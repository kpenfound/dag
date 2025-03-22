from typing import Annotated
from firecrawl import FirecrawlApp
import dagger
from dagger import Doc, function, object_type

@object_type
class CrawlPage:
    url: str
    title: str
    description: str
    content: str

@object_type
class FirecrawlDag:
    api_key: dagger.Secret

    @function
    async def crawl(
        self,
        url: Annotated[str, Doc("The URL to crawl.")],
    ) -> list[CrawlPage]:
        """Crawl an entire website and return the data containing all of its pages."""
        app = FirecrawlApp(api_key=await self.api_key.plaintext())

        # Crawl a website:
        crawl_response = app.crawl_url(
          url,
          params={
            'limit': 100,
            'scrapeOptions': {'formats': ['markdown']}
          },
          poll_interval=30
        )
        pages = []
        for page in crawl_response['data']:
            pages.append(CrawlPage(
                url=page['metadata']['sourceURL'],
                title=page['metadata']['title'],
                description=page['metadata']['description'],
                content=page['markdown']
            ))
        return pages

    @function
    async def scrape(
        self,
        url: Annotated[str, Doc("The URL to scrape.")],
    ) -> str:
        """Scrape a signle webpage and return the content in markdown format."""
        app = FirecrawlApp(api_key=await self.api_key.plaintext())

        # Scrape a website:
        scrape_result = app.scrape_url(url, params={'formats': ['markdown']})
        return scrape_result["markdown"]

    @function
    async def map(
        self,
        url: Annotated[str, Doc("The URL to scrape.")],
    ) -> list[str]:
        """Map a website to get a list of all the page urls"""
        app = FirecrawlApp(api_key=await self.api_key.plaintext())

        # Map a website:
        map_result = app.map_url(url)
        return map_result["links"]

    # @function
    # async def extract(
    #     self,
    #     urls: Annotated[list[str], Doc("The URLs to scrape.")],
    #     prompt: Annotated[str, Doc("The description of the information to extract.")],
    # ) -> str:
    #     """Extract structured data from a website"""
    #     # app = FirecrawlApp(api_key=await self.api_key.plaintext())
    #     # FIXME: The ExtractParams seems to be different from the docs
    #     # data = app.extract(urls, {
    #     #     'prompt': prompt,
    #     # })
    #     # return data["data"]
    #     return "Not implemented"
