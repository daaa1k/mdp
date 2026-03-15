class Mdpaste < Formula
  desc "Paste clipboard image as Markdown link"
  homepage "https://github.com/daaa1k/mdpaste"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/daaa1k/mdpaste/releases/download/v#{version}/mdpaste-macos-aarch64"
      sha256 "" # fill in after first release

      def install
        bin.install "mdpaste-macos-aarch64" => "mdpaste"
      end
    else
      url "https://github.com/daaa1k/mdpaste/releases/download/v#{version}/mdpaste-macos-x86_64"
      sha256 "" # fill in after first release

      def install
        bin.install "mdpaste-macos-x86_64" => "mdpaste"
      end
    end
  end

  on_linux do
    url "https://github.com/daaa1k/mdpaste/releases/download/v#{version}/mdpaste-linux-x86_64"
    sha256 "" # fill in after first release

    def install
      bin.install "mdpaste-linux-x86_64" => "mdpaste"
    end
  end

  test do
    system "#{bin}/mdpaste", "--help"
  end
end
