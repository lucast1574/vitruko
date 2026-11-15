import React from "react";
import {
  Html,
  Head,
  Preview,
  Body,
  Container,
  Section,
  Heading,
  Text,
  Button,
  Hr,
  Link,
  Img,
  Row,
  Column,
} from "react-email";
const h = React.createElement;
export function Transactional({ brand, site, accent, background, tagline }) {
  return h(
    Html,
    { lang: "es" },
    h(
      Head,
      null,
      h("meta", {
        name: "viewport",
        content: "width=device-width, initial-scale=1",
      }),
    ),
    h(Preview, null, "{{.Preview}}"),
    h(
      Body,
      {
        lang: "es",
        style: {
          backgroundColor: background,
          fontFamily: "Arial, Helvetica, sans-serif",
          margin: 0,
          padding: "32px 12px",
          color: "#172c29",
        },
      },
      h(
        Container,
        { style: { maxWidth: "560px", margin: "0 auto" } },
        h(
          Section,
          { style: { padding: "8px 8px 24px" } },
          h(
            Row,
            null,
            h(
              Column,
              { style: { width: "44px" } },
              h(Img, {
                src: site + "/email-logo.png",
                alt: "",
                width: "36",
                height: "36",
                style: { display: "block", border: 0 },
              }),
            ),
            h(
              Column,
              null,
              h(
                Text,
                {
                  style: {
                    fontSize: "27px",
                    fontWeight: 700,
                    letterSpacing: "-1px",
                    margin: 0,
                    color: accent,
                  },
                },
                brand.toLowerCase() + ".",
              ),
            ),
          ),
        ),
        h(
          Section,
          {
            style: {
              backgroundColor: "#ffffff",
              padding: "36px 28px",
              borderRadius: "16px",
            },
          },
          h(
            Text,
            {
              style: {
                color: accent,
                fontSize: "13px",
                fontWeight: 700,
                margin: "0 0 18px",
              },
            },
            "{{.Label}}",
          ),
          h(
            Heading,
            {
              as: "h1",
              style: {
                fontSize: "30px",
                lineHeight: "36px",
                letterSpacing: "-1px",
                fontWeight: 700,
                margin: "0 0 24px",
              },
            },
            "{{.Title}}",
          ),
          h(
            Text,
            {
              style: {
                fontSize: "16px",
                lineHeight: "26px",
                margin: "0 0 16px",
              },
            },
            "Hola, {{.Name}}.",
          ),
          h(
            Text,
            {
              style: {
                fontSize: "16px",
                lineHeight: "26px",
                margin: "0 0 16px",
              },
            },
            "{{.Body}}",
          ),
          h(
            Section,
            {
              style: {
                backgroundColor: background,
                borderRadius: "10px",
                padding: "16px 20px",
                margin: "24px 0",
              },
            },
            h(
              Text,
              { style: { fontSize: "14px", lineHeight: "23px", margin: 0 } },
              "{{.Detail}}",
            ),
          ),
          h(
            Button,
            {
              href: "{{.ActionURL}}",
              style: {
                backgroundColor: accent,
                color: "#ffffff",
                padding: "15px 24px",
                fontSize: "15px",
                fontWeight: 700,
                borderRadius: "8px",
                textDecoration: "none",
              },
            },
            "{{.Action}}",
          ),
          h(Hr, { style: { borderColor: "#e5ebe8", margin: "30px 0 20px" } }),
          h(
            Text,
            {
              style: {
                fontSize: "13px",
                lineHeight: "21px",
                color: "#687571",
                margin: 0,
              },
            },
            "{{.Footnote}}",
          ),
        ),
        h(
          Text,
          {
            style: {
              fontSize: "12px",
              lineHeight: "20px",
              color: "#687571",
              padding: "8px",
              textAlign: "center",
            },
          },
          brand + " · " + tagline,
          h("br"),
          h(
            Link,
            {
              href: site,
              style: { color: accent, textDecoration: "underline" },
            },
            site.replace("https://", ""),
          ),
        ),
      ),
    ),
  );
}
