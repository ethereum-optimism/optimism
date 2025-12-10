import React from 'react';
import clsx from 'clsx';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import styles from './index.module.css';
import useBaseUrl from '@docusaurus/useBaseUrl';

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <h1 className="hero__title">Welcome to the Boba docs!</h1>
        <h3>
          Something snappy here
        </h3>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <div className={styles.buttons}>
          <Link className="button button--secondary button--lg" to="/dev-docs/index">
            Developer Docs
          </Link>
          &nbsp; &nbsp; &nbsp;
          <Link className="button button--secondary button--lg" to="/user-docs/index">
            User Docs
          </Link>
        </div>
      </div>
    </header>
  );
}

export default function Home() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <Layout title={siteConfig.title} description="Boba Documentation Home page">
      <HomepageHeader />
      <main>
        <div className={styles.section}>
          <div className="container text--center margin-bottom--xl">
            <div className="row">
              <div className="col">
                <img
                  className={styles.featureImage}
                  alt="dev image"
                  src={useBaseUrl('/img/platonic-icons/platonic-platform.svg')}
                />
                <h2 className={clsx(styles.featureHeading)}>Developer</h2>
                <p className="padding-horiz--md">
                  The developer docs are pretty cool
                </p>
              </div>
              <div className="col">
                <img
                  alt="user image"
                  className={styles.featureImage}
                  src={useBaseUrl('/img/platonic-icons/SDK.svg')}
                />
                <h2 className={clsx(styles.featureHeading)}>User</h2>
                <p className="padding-horiz--md">
                  The user docs are cool too, I guess
                </p>
              </div>
              <div hidden={true}>
                Icons made by{' '}
                <a href="https://www.freepik.com" title="Freepik">
                  Freepik
                </a>{' '}
                from{' '}
                <a href="https://www.flaticon.com/" title="Flaticon">
                  www.flaticon.com
                </a>
              </div>
            </div>
            </div>
        </div>
      </main>
    </Layout>
  );
}
