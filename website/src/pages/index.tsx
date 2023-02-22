import React from 'react';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';

export default function Home(): JSX.Element {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={siteConfig.title}
      description={siteConfig.tagline}>
      <Hero />
    </Layout>
  );
}

export function Hero() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <div className="grow grid bg-indigo-700">
      <div className="m-auto max-w-2xl py-16 px-6 text-center sm:py-20 lg:px-8">
        <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
          <span className="block">{siteConfig.title}</span>
          <span className="block">{siteConfig.tagline}</span>
        </h2>
        <p className="mt-4 text-lg leading-6 text-indigo-200">
          Transform the way you work with SQL Databases in Go
        </p>
        <a
          href="docs"
          className="mt-8 inline-flex w-full items-center justify-center rounded-md border border-transparent bg-white px-5 py-3 text-base font-medium text-indigo-600 hover:bg-indigo-50 sm:w-auto"
        >
          Get Started
        </a>
      </div>
    </div>
  )
}

